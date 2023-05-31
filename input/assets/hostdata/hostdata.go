// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package hostdata

import (
	"context"
	"sync"
	"time"

	input "github.com/elastic/beats/v7/filebeat/input/v2"
	stateless "github.com/elastic/beats/v7/filebeat/input/v2/input-stateless"
	"github.com/elastic/beats/v7/libbeat/feature"
	"github.com/elastic/elastic-agent-system-metrics/metric/system/host"
	"github.com/elastic/go-sysinfo"
	"github.com/elastic/inputrunner/input/assets/internal"

	"github.com/elastic/beats/v7/libbeat/processors/add_host_metadata"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/elastic-agent-libs/monitoring"

	"github.com/elastic/go-concert/ctxtool"
)

var (
	reg *monitoring.Registry
)

type config struct {
	internal.BaseConfig `config:",inline"`
	KubeConfig          string        `config:"kube_config"`
	Period              time.Duration `config:"period"`
}

type addHostMetadata struct {
	lastUpdate struct {
		time.Time
		sync.Mutex
	}
	data    mapstr.Pointer
	geoData mapstr.M
	config  add_host_metadata.Config
	logger  *logp.Logger
}

func defaultHostMetadataConfig() add_host_metadata.Config {
	return add_host_metadata.Config{
		NetInfoEnabled: true,
		CacheTTL:       5 * time.Minute,
		ReplaceFields:  true,
	}
}

func Plugin() input.Plugin {
	return input.Plugin{
		Name:       "assets_hostdata",
		Stability:  feature.Stable,
		Deprecated: false,
		Info:       "assets_hostdata",
		Manager:    stateless.NewInputManager(configure),
	}
}

func configure(inputCfg *conf.C) (stateless.Input, error) {
	cfg := defaultConfig()
	if err := inputCfg.Unpack(&cfg); err != nil {
		return nil, err
	}
	reg = monitoring.Default.NewRegistry("hostdata", monitoring.DoNotReport)
	return newAssetsHostdata(cfg)
}

func newAssetsHostdata(cfg config) (*assetsHostdata, error) {
	return &assetsHostdata{cfg}, nil
}

func defaultConfig() config {
	return config{
		BaseConfig: internal.BaseConfig{
			Period:     time.Second * 600,
			AssetTypes: nil,
		},
		Period: time.Second * 600,
	}
}

type assetsHostdata struct {
	Config config
}

func (s *assetsHostdata) Name() string { return "assets_hostdata" }

func (s *assetsHostdata) Test(_ input.TestContext) error {
	return nil
}

func (s *assetsHostdata) Run(inputCtx input.Context, publisher stateless.Publisher) error {
	ctx := ctxtool.FromCanceller(inputCtx.Cancelation)
	log := inputCtx.Logger.With("assets_hostdata")

	log.Info("hostdata asset collector run started")
	defer log.Info("hostdata asset collector run stopped")

	cfg := s.Config
	ticker := time.NewTicker(cfg.Period)
	collectHostdataAssets(ctx, log, cfg, publisher)
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			collectHostdataAssets(ctx, log, cfg, publisher)
		}
	}
}

// collectHostdataAssets collects kubernetes resources from watchers cache and publishes them
func collectHostdataAssets(ctx context.Context, log *logp.Logger, cfg config, publisher stateless.Publisher) {
	indexNamespace := cfg.IndexNamespace
	go func() {
		c := defaultHostMetadataConfig()
		p := &addHostMetadata{
			config: c,
			data:   mapstr.NewPointer(nil),
			logger: log,
		}
		err := p.getData(log)
		if err != nil {
			log.Errorf("error loading data: %w", err)
		}
		log.Infof("Dataaaaaaaa %+w", p.data.Get())
		publishHostdata(ctx, log, indexNamespace, publisher, p.data.Get())
	}()
}

func (p *addHostMetadata) expired() bool {
	if p.config.CacheTTL <= 0 {
		return true
	}

	p.lastUpdate.Lock()
	defer p.lastUpdate.Unlock()

	if p.lastUpdate.Add(p.config.CacheTTL).After(time.Now()) {
		return false
	}
	p.lastUpdate.Time = time.Now()
	return true
}

func (p *addHostMetadata) getData(log *logp.Logger) error {
	if !p.expired() {
		return nil
	}

	h, err := sysinfo.Host()
	if err != nil {
		return err
	}

	data := host.MapHostInfo(h.Info())
	//log.Infof("DAAAAATA %+w", data)

	p.data.Set(data)
	return nil
}

// publishHostdata publishes the host assets
func publishHostdata(ctx context.Context, log *logp.Logger, indexNamespace string, publisher stateless.Publisher, data mapstr.M) {
	log.Info("Publishing host assets\n")
	assetType := "host"
	assetKind := "host"
	hostdata, _ := data.GetValue("host")
	hostdataMap := hostdata.(mapstr.M)
	hostname, _ := hostdataMap.GetValue("hostname")
	assetId, _ := hostdataMap.GetValue("id")
	architecture, _ := hostdataMap.GetValue("architecture")
	osData, _ := hostdataMap.GetValue("os")
	osDataMap := osData.(mapstr.M)
	osBuild, _ := osDataMap.GetValue("build")
	osFamily, _ := osDataMap.GetValue("family")
	osKernel, _ := osDataMap.GetValue("kernel")
	osName, _ := osDataMap.GetValue("name")
	osPlatform, _ := osDataMap.GetValue("platform")
	osType, _ := osDataMap.GetValue("type")
	osVersion, _ := osDataMap.GetValue("version")
	log.Debugf("Publish Host: %+v", hostname)
	log.Debug("Host  id: ", assetId)
	options := []internal.AssetOption{
		internal.WithAssetTypeAndID(assetType, assetId.(string)),
		internal.WithAssetKind(assetKind),
		internal.WithHostData(hostname.(string), architecture.(string)),
		internal.WithHostOsData(osBuild.(string), osFamily.(string), osKernel.(string), osName.(string), osPlatform.(string), osType.(string), osVersion.(string)),
		internal.WithIndex(assetType, indexNamespace),
	}

	internal.Publish(publisher, options...)

}

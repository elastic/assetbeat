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
	"fmt"
	"time"

	"github.com/elastic/assetbeat/input/internal"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/processors/add_host_metadata"
	stateless "github.com/elastic/beats/v7/filebeat/input/v2/input-stateless"
	input "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/libbeat/feature"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/go-concert/ctxtool"
)

func Plugin() input.Plugin {
	return input.Plugin{
		Name:       "hostdata",
		Stability:  feature.Stable,
		Deprecated: false,
		Info:       "hostdata",
		Manager:    stateless.NewInputManager(configure),
	}
}

type config struct {
	internal.BaseConfig `config:",inline"`
}

type hostdata struct {
	config                   config
	addHostMetadataProcessor beat.Processor
}

func configure(inputCfg *conf.C) (stateless.Input, error) {
	cfg := config{
		BaseConfig: internal.BaseConfig{
			Period: time.Minute,
		},
	}
	if err := inputCfg.Unpack(&cfg); err != nil {
		return nil, fmt.Errorf("error unpacking config: %w", err)
	}

	return newHostdata(cfg)
}

func newHostdata(cfg config) (*hostdata, error) {
	processor, err := add_host_metadata.New(conf.NewConfig())
	if err != nil {
		return nil, fmt.Errorf("error creating host metadata processor: %w", err)
	}

	return &hostdata{
		config:                   cfg,
		addHostMetadataProcessor: processor,
	}, nil
}

func (h *hostdata) Name() string { return "hostdata" }

func (h *hostdata) Test(_ input.TestContext) error {
	return nil
}

func (h *hostdata) Run(inputCtx input.Context, publisher stateless.Publisher) error {
	ctx := ctxtool.FromCanceller(inputCtx.Cancelation)
	logger := inputCtx.Logger

	logger.Info("hostdata asset collector run started")
	defer logger.Info("hostdata asset collector run stopped")

	ticker := time.NewTicker(h.config.Period)
	select {
	case <-ctx.Done():
		return nil
	default:
		h.collectHostdataAssets(ctx, logger, publisher)
	}
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			h.collectHostdataAssets(ctx, logger, publisher)
		}
	}
}

func (h *hostdata) collectHostdataAssets(_ context.Context, logger *logp.Logger, publisher stateless.Publisher) {
	logger.Debug("collecting hostdata asset information")

	// we make use of libbeat's add_host_metadata processor to populate all the necessary
	// ECS host fields in the event, then add the required asset fields.
	event := internal.NewEvent()
	event, err := h.addHostMetadataProcessor.Run(event)
	if err != nil {
		logger.Error("error collecting hostdata: %w", err)
		return
	}

	hostID, err := event.Fields.GetValue("host.id")
	if err != nil {
		logger.Error("no host ID in collected hostdata")
		return
	}

	assetKind := "host"
	assetType := "host"
	internal.Publish(publisher, event,
		internal.WithAssetKindAndID(assetKind, hostID.(string)),
		internal.WithAssetType(assetType),
		internal.WithIndex(assetType, h.config.IndexNamespace),
	)
}

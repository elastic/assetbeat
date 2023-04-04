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

package k8s

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/elastic/inputrunner/input/assets/internal"
	input "github.com/elastic/inputrunner/input/v2"
	stateless "github.com/elastic/inputrunner/input/v2/input-stateless"

	kube "github.com/elastic/elastic-agent-autodiscover/kubernetes"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"

	"github.com/elastic/beats/v7/libbeat/feature"
	"github.com/elastic/go-concert/ctxtool"

	kuberntescli "k8s.io/client-go/kubernetes"
)

type config struct {
	internal.BaseConfig `config:",inline"`
	KubeConfig          string        `config:"kube_config"`
	Period              time.Duration `config:"period"`
}

type watchersMap struct {
	watchers sync.Map
}

func Plugin() input.Plugin {
	return input.Plugin{
		Name:       "assets_k8s",
		Stability:  feature.Stable,
		Deprecated: false,
		Info:       "assets_k8s",
		Manager:    stateless.NewInputManager(configure),
	}
}

func configure(inputCfg *conf.C) (stateless.Input, error) {
	cfg := defaultConfig()
	if err := inputCfg.Unpack(&cfg); err != nil {
		return nil, err
	}

	return newAssetsK8s(cfg)
}

func newAssetsK8s(cfg config) (*assetsK8s, error) {
	return &assetsK8s{cfg}, nil
}

func defaultConfig() config {
	return config{
		BaseConfig: internal.BaseConfig{
			Period:     time.Second * 600,
			AssetTypes: nil,
		},
		KubeConfig: "",
		Period:     time.Second * 600,
	}
}

type assetsK8s struct {
	Config config
}

func (s *assetsK8s) Name() string { return "assets_k8s" }

func (s *assetsK8s) Test(_ input.TestContext) error {
	return nil
}

func (s *assetsK8s) Run(inputCtx input.Context, publisher stateless.Publisher) error {
	ctx := ctxtool.FromCanceller(inputCtx.Cancelation)
	log := inputCtx.Logger.With("assets_k8s")

	log.Info("k8s asset collector run started")
	defer log.Info("k8s asset collector run stopped")

	cfg := s.Config
	kubeConfigPath := cfg.KubeConfig
	ticker := time.NewTicker(cfg.Period)

	client, err := getKubernetesClient(kubeConfigPath, log)
	if err != nil {
		log.Errorf("unable to build kubernetes clientset: %w", err)
	}

	// var watchers map[string]kube.Watcher
	watchersMap := &watchersMap{}
	select {
	case <-ctx.Done():
		return nil
	default:
		initK8sWatchers(ctx, client, log, cfg, publisher, watchersMap)
		collectK8sAssets(ctx, log, cfg, publisher, watchersMap)
	}
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			collectK8sAssets(ctx, log, cfg, publisher, watchersMap)
		}
	}
	return nil

}

// getKubernetesClient returns a kubernetes client. If inCluster is true, it returns an
// in cluster configuration based on the secrets mounted in the Pod. If kubeConfig is passed,
// it parses the config file to get the config required to build a client.
func getKubernetesClient(kubeconfigPath string, log *logp.Logger) (kuberntescli.Interface, error) {
	log.Infof("Provided kube config path is %s", kubeconfigPath)
	cfg, err := kube.BuildConfig(kubeconfigPath)
	if err != nil {
		return nil, fmt.Errorf("unable to build kubernetes config: %w", err)
	}

	client, err := kuberntescli.NewForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("unable to build kubernetes client: %w", err)
	}

	return client, nil
}

// collectK8sAssets initiates watchers for kubernetes nodes and pods, which watch for resources in kubernetes cluster
func collectK8sAssets(ctx context.Context, log *logp.Logger, cfg config, publisher stateless.Publisher, watchersMap *watchersMap) {
	if internal.IsTypeEnabled(cfg.AssetTypes, "node") {
		log.Info("Node type enabled. Starting collecting")
		go func() {
			if nodeWatcher, ok := watchersMap.watchers.Load("node"); ok {
				publishK8sNodes(log, publisher, nodeWatcher.(kube.Watcher))
			} else {
				log.Error("Node watcher not found")
			}

		}()
	}
	if internal.IsTypeEnabled(cfg.AssetTypes, "pod") {
		log.Info("Pod type enabled. Starting collecting")
		go func() {
			if podWatcher, ok := watchersMap.watchers.Load("pod"); ok {
				if internal.IsTypeEnabled(cfg.AssetTypes, "node") {
					if nodeWatcher, ok := watchersMap.watchers.Load("node"); ok {
						publishK8sPods(log, publisher, podWatcher.(kube.Watcher), nodeWatcher.(kube.Watcher))
					} else {
						publishK8sPods(log, publisher, podWatcher.(kube.Watcher), nil)
					}
				} else {
					publishK8sPods(log, publisher, podWatcher.(kube.Watcher), nil)
				}
			} else {
				log.Error("Pod watcher not found")
			}

		}()
	}
}

// initK8sWatchers initiates watchers for kubernetes nodes and pods, which watch for resources in kubernetes cluster
func initK8sWatchers(ctx context.Context, client kuberntescli.Interface, log *logp.Logger, cfg config, publisher stateless.Publisher, watchersMap *watchersMap) {

	if internal.IsTypeEnabled(cfg.AssetTypes, "node") {
		log.Info("Node type enabled. Initiate node watcher")
		go func() {
			nodeWatcher, err := watchK8sNodes(ctx, log, client, time.Second*60)
			if err != nil {
				log.Errorf("error initiating Node watcher: %w", err)
			}
			watchersMap.watchers.Store("node", nodeWatcher)
		}()
	}
	if internal.IsTypeEnabled(cfg.AssetTypes, "pod") {
		log.Info("Pod type enabled. Initiate pod watcher")
		go func() {
			podWatcher, err := watchK8sPods(ctx, log, client, time.Second*60)
			if err != nil {
				log.Errorf("error initiating Pod watcher: %w", err)
			}
			watchersMap.watchers.Store("pod", podWatcher)
		}()
	}
}

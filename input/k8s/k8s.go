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

	input "github.com/elastic/beats/v7/filebeat/input/v2"
	stateless "github.com/elastic/beats/v7/filebeat/input/v2/input-stateless"
	"github.com/elastic/inputrunner/input/internal"

	kube "github.com/elastic/elastic-agent-autodiscover/kubernetes"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"

	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"

	"github.com/elastic/beats/v7/libbeat/feature"
	"github.com/elastic/go-concert/ctxtool"

	kuberntescli "k8s.io/client-go/kubernetes"
)

type config struct {
	internal.BaseConfig `config:",inline"`
	KubeConfig          string        `config:"kube_config"`
	Period              time.Duration `config:"period"`
}

// watchersMap struct containt a sync.Map object to effectively handle
// concurrent writes and reads of the map values
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
	log := logp.NewLogger("assets_k8s")
	client, err := getKubernetesClient(cfg.KubeConfig, log)
	if err != nil {
		log.Errorf("unable to build kubernetes clientset: %w", err)
	}

	return newAssetsK8s(cfg, client)
}

func newAssetsK8s(cfg config, client kuberntescli.Interface) (*assetsK8s, error) {
	return &assetsK8s{cfg, client}, nil
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
	Client kuberntescli.Interface
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
	ticker := time.NewTicker(cfg.Period)

	client := s.Client
	if client == nil {
		return fmt.Errorf("Kubernetes client is nil")
	}

	watchersMap := &watchersMap{}
	select {
	case <-ctx.Done():
		return nil
	default:
		// Init the watchers
		if err := initK8sWatchers(ctx, client, log, cfg, publisher, watchersMap); err != nil {
			return err
		}
		// Start the watchers
		if err := startK8sWatchers(ctx, log, cfg, watchersMap); err != nil {
			// stop any running watcher
			stopK8sWatchers(ctx, log, watchersMap)
			return err
		}
		// wait 10 seconds for cache to be filled. Only applicable on first run
		time.AfterFunc(10*time.Second, func() { collectK8sAssets(ctx, log, cfg, publisher, watchersMap) })
	}
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			collectK8sAssets(ctx, log, cfg, publisher, watchersMap)
		}
	}
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

// collectK8sAssets collects kubernetes resources from watchers cache and publishes them
func collectK8sAssets(ctx context.Context, log *logp.Logger, cfg config, publisher stateless.Publisher, watchersMap *watchersMap) {
	indexNamespace := cfg.IndexNamespace
	if internal.IsTypeEnabled(cfg.AssetTypes, "k8s.node") {
		log.Info("Node type enabled. Starting collecting")
		go func() {
			if nodeWatcher, ok := watchersMap.watchers.Load("node"); ok {
				nw, ok := nodeWatcher.(kube.Watcher)
				if ok {
					publishK8sNodes(ctx, log, indexNamespace, publisher, nw, kube.IsInCluster(cfg.KubeConfig))
				} else {
					log.Error("Node watcher type assertion failed")
				}
			} else {
				log.Error("Node watcher not found")
			}

		}()
	}
	if internal.IsTypeEnabled(cfg.AssetTypes, "k8s.pod") {
		log.Info("Pod type enabled. Starting collecting")
		go func() {
			if podWatcher, ok := watchersMap.watchers.Load("pod"); ok {
				var nw kube.Watcher
				if internal.IsTypeEnabled(cfg.AssetTypes, "k8s.node") {
					if nodeWatcher, ok := watchersMap.watchers.Load("node"); ok {
						nw, ok = nodeWatcher.(kube.Watcher)
						if !ok {
							log.Error("Node watcher type assertion failed")
						}
					}
				}
				pw, ok := podWatcher.(kube.Watcher)
				if ok {
					publishK8sPods(ctx, log, indexNamespace, publisher, pw, nw)
				} else {
					log.Error("Pod watcher type assertion failed")
				}

			} else {
				log.Error("Pod watcher not found")
			}

		}()
	}

	if internal.IsTypeEnabled(cfg.AssetTypes, "k8s.container") {
		log.Info("Container type enabled. Starting collecting")
		go func() {
			if podWatcher, ok := watchersMap.watchers.Load("pod"); ok {
				pw, ok := podWatcher.(kube.Watcher)
				if ok {
					publishK8sContainers(ctx, log, indexNamespace, publisher, pw)
				} else {
					log.Error("Pod watcher type assertion failed")
				}

			} else {
				log.Error("Pod watcher not found")
			}

		}()
	}
}

// initK8sWatchers initiates and stores watchers for kubernetes nodes and pods, which watch for resources in kubernetes cluster
func initK8sWatchers(ctx context.Context, client kuberntescli.Interface, log *logp.Logger, cfg config, publisher stateless.Publisher, watchersMap *watchersMap) error {

	if internal.IsTypeEnabled(cfg.AssetTypes, "k8s.node") {
		log.Info("Node type enabled. Initiate node watcher")
		nodeWatcher, err := getNodeWatcher(ctx, log, client, time.Second*60)
		if err != nil {
			log.Errorf("error initiating Node watcher: %w", err)
			return err
		}
		watchersMap.watchers.Store("node", nodeWatcher)
	}

	if internal.IsTypeEnabled(cfg.AssetTypes, "k8s.pod") {
		log.Info("Pod type enabled. Initiate pod watcher")
		podWatcher, err := getPodWatcher(ctx, log, client, time.Second*60)
		if err != nil {
			log.Errorf("error initiating Pod watcher: %w", err)
			return err
		}
		watchersMap.watchers.Store("pod", podWatcher)
	}
	return nil
}

// startK8sWatchers starts the given watchers
func startK8sWatchers(ctx context.Context, log *logp.Logger, cfg config, watchersMap *watchersMap) error {

	if internal.IsTypeEnabled(cfg.AssetTypes, "k8s.node") {
		log.Info("Starting node watcher")
		if nodeWatcher, ok := watchersMap.watchers.Load("node"); ok {
			nw, ok := nodeWatcher.(kube.Watcher)
			if ok {
				if err := nw.Start(); err != nil {
					log.Errorf("Couldn't start node watcher: %v", err)
					return err
				}
			} else {
				return fmt.Errorf("node watcher type assertion failed")
			}
		} else {
			return fmt.Errorf("node watcher not found")
		}
	}

	if internal.IsTypeEnabled(cfg.AssetTypes, "k8s.pod") {
		log.Info("Starting pod watcher")
		if podWatcher, ok := watchersMap.watchers.Load("pod"); ok {
			pw, ok := podWatcher.(kube.Watcher)
			if ok {
				if err := pw.Start(); err != nil {
					log.Errorf("Couldn't start pod watcher: %v", err)
					return err
				}
			} else {
				return fmt.Errorf("pod watcher type assertion failed")
			}
		} else {
			return fmt.Errorf("pod watcher not found")
		}
	}

	return nil
}

// stopK8sWatchers starts the given watchers
func stopK8sWatchers(ctx context.Context, log *logp.Logger, watchersMap *watchersMap) {

	log.Info("Stoping watchers")
	if podWatcher, ok := watchersMap.watchers.Load("pod"); ok {
		pw, ok := podWatcher.(kube.Watcher)
		if ok {
			pw.Stop()
		} else {
			log.Error("pod watcher type assertion failed")
		}
	} else {
		log.Error("pod watcher not found")
	}

	if nodeWatcher, ok := watchersMap.watchers.Load("node"); ok {
		nw, ok := nodeWatcher.(kube.Watcher)
		if ok {
			nw.Stop()
		} else {
			log.Error("node watcher type assertion failed")
		}
	} else {
		log.Error("node watcher not found")
	}
}

// SetClient sets the Kubernetes Client. Used for e2e tests
func SetClient(client kuberntescli.Interface, s stateless.Input) error {
	i, ok := s.(*assetsK8s)
	if !ok {
		return fmt.Errorf("stateless.Input to assetsK8s type assertion failed")
	}
	i.Client = client
	return nil
}

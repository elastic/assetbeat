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

package assets_k8s

import (
	"context"
	"fmt"
	"time"

	input "github.com/elastic/inputrunner/input/v2"
	stateless "github.com/elastic/inputrunner/input/v2/input-stateless"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/feature"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/go-concert/ctxtool"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

func Plugin() input.Plugin {
	return input.Plugin{
		Name:       "assets_k8s",
		Stability:  feature.Stable,
		Deprecated: false,
		Info:       "assets_k8s",
		Manager:    stateless.NewInputManager(configure),
	}
}

func configure(cfg *conf.C) (stateless.Input, error) {
	config := defaultConfig()
	if err := cfg.Unpack(&config); err != nil {
		return nil, err
	}

	return newAssetsK8s(config)
}

func newAssetsK8s(config config) (*assetsK8s, error) {
	return &assetsK8s{config}, nil
}

type Config struct {
	KubeConfig string        `config:"kube_config"`
	Period     time.Duration `config:"period"`
}

func defaultConfig() config {
	return config{
		Config: Config{
			KubeConfig: "",
			Period:     time.Second * 600,
		},
	}
}

type assetsK8s struct {
	config
}

type config struct {
	Config `config:",inline"`
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
	kubeConfigPath := s.config.Config.KubeConfig

	ticker := time.NewTicker(s.config.Config.Period)
	select {
	case <-ctx.Done():
		return nil
	default:
		collectK8sAssets(ctx, kubeConfigPath, log, publisher)
	}
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			collectK8sAssets(ctx, kubeConfigPath, log, publisher)
		}
	}
}

// getKubernetesClient returns a kubernetes client. If inCluster is true, it returns an
// in cluster configuration based on the secrets mounted in the Pod. If kubeConfig is passed,
// it parses the config file to get the config required to build a client.
func getKubernetesClient(kubeconfigPath string) (kubernetes.Interface, error) {

	cfg, err := BuildConfig(kubeconfigPath)
	if err != nil {
		return nil, fmt.Errorf("unable to build kubernetes config: %w", err)
	}

	client, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("unable to build kubernetes client: %w", err)
	}

	return client, nil
}

func collectK8sAssets(ctx context.Context, kubeconfigPath string, log *logp.Logger, publisher stateless.Publisher) {

	client, err := getKubernetesClient(kubeconfigPath)
	if err != nil {
		log.Errorf("unable to build kubernetes clientset: %w", err)
	}
	go collectK8sNodes(ctx, log, client, publisher)
}

// collect the kubernetes nodes
func collectK8sNodes(ctx context.Context, log *logp.Logger, client kubernetes.Interface, publisher stateless.Publisher) error {

	// collect the nodes using the client
	nodes, _ := client.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})

	for _, node := range nodes.Items {
		log.Infof("%s\n", node.Name)
		assetId := node.Spec.ProviderID
		publishK8sAsset(node.Name, "k8s.node", assetId, publisher)
	}
	return nil
}

// publishes the kubernetes assets
func publishK8sAsset(name string, assetType, assetId string, publisher stateless.Publisher) {

	asset := mapstr.M{
		"asset.name": name,
		"asset.type": assetType,
		"asset.id":   assetId,
		"asset.ean":  fmt.Sprintf("%s:%s", assetType, assetId),
	}

	publisher.Publish(beat.Event{Fields: asset})

}

// BuildConfig is a helper function that builds configs from a kubeconfig filepath.
// If kubeconfigPath is not passed in we fallback to inClusterConfig.
// If inClusterConfig fails, we fallback to the default config.
// This is a copy of `clientcmd.BuildConfigFromFlags` of `client-go` but without the annoying
// klog messages that are not possible to be disabled.
func BuildConfig(kubeconfigPath string) (*restclient.Config, error) {
	if kubeconfigPath == "" {
		kubeconfig, err := restclient.InClusterConfig()
		if err == nil {
			return kubeconfig, nil
		}
	}
	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeconfigPath},
		&clientcmd.ConfigOverrides{ClusterInfo: clientcmdapi.Cluster{Server: ""}}).ClientConfig()
}

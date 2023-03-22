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
	"errors"
	"fmt"
	"time"

	"github.com/elastic/inputrunner/input/assets/internal"
	input "github.com/elastic/inputrunner/input/v2"
	stateless "github.com/elastic/inputrunner/input/v2/input-stateless"

	util "github.com/elastic/elastic-agent-autodiscover/kubernetes"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"

	"github.com/elastic/beats/v7/libbeat/feature"
	"github.com/elastic/go-concert/ctxtool"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type config struct {
	internal.BaseConfig `config:",inline"`
	KubeConfig          string        `config:"kube_config"`
	Period              time.Duration `config:"period"`
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
	select {
	case <-ctx.Done():
		return nil
	default:
		collectK8sAssets(ctx, kubeConfigPath, log, cfg, publisher)
	}
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			collectK8sAssets(ctx, kubeConfigPath, log, cfg, publisher)
		}
	}
}

// getKubernetesClient returns a kubernetes client. If inCluster is true, it returns an
// in cluster configuration based on the secrets mounted in the Pod. If kubeConfig is passed,
// it parses the config file to get the config required to build a client.
func getKubernetesClient(kubeconfigPath string, log *logp.Logger) (kubernetes.Interface, error) {
	log.Infof("Provided kube config path is %s", kubeconfigPath)
	cfg, err := util.BuildConfig(kubeconfigPath)
	if err != nil {
		return nil, fmt.Errorf("unable to build kubernetes config: %w", err)
	}

	client, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("unable to build kubernetes client: %w", err)
	}

	return client, nil
}

func collectK8sAssets(ctx context.Context, kubeconfigPath string, log *logp.Logger, cfg config, publisher stateless.Publisher) {

	client, err := getKubernetesClient(kubeconfigPath, log)
	if err != nil {
		log.Errorf("unable to build kubernetes clientset: %w", err)
	}

	if internal.IsTypeEnabled(cfg.AssetTypes, "node") {
		log.Info("Node type enabled. Starting collecting")
		go func() {
			err := collectK8sNodes(ctx, log, client, publisher)
			if err != nil {
				log.Errorf("error collecting Node assets: %w", err)
			}
		}()
	}
	if internal.IsTypeEnabled(cfg.AssetTypes, "pod") {
		log.Info("Pod type enabled. Starting collecting")
		go func() {
			err := collectK8sPods(ctx, log, client, publisher)
			if err != nil {
				log.Errorf("error collecting Pod assets: %w", err)
			}
		}()
	}
}

// collect the kubernetes nodes
func collectK8sNodes(ctx context.Context, log *logp.Logger, client kubernetes.Interface, publisher stateless.Publisher) error {

	// collect the nodes using the client
	nodes, err := client.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		log.Errorf("Cannot list k8s nodes: %w", err)
		return err
	}

	log.Info("Started collecting nodes information\n")
	for _, node := range nodes.Items {
		assetProviderId := node.Spec.ProviderID
		assetId := string(node.ObjectMeta.UID)
		assetStartTime := node.ObjectMeta.CreationTimestamp
		assetParents := []string{}

		log.Info("Publishing nodes assets\n")
		internal.Publish(publisher,
			internal.WithAssetTypeAndID("k8s.node", assetId),
			internal.WithAssetParents(assetParents),
			internal.WithNodeData(node.Name, assetProviderId, &assetStartTime),
		)
	}
	return nil
}

func collectK8sPods(ctx context.Context, log *logp.Logger, client kubernetes.Interface, publisher stateless.Publisher) error {
	pods, err := client.CoreV1().Pods("").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		log.Errorf("Cannot list k8s pods: %w", err)
		return err
	}

	log.Info("Started collecting pods information\n")

	for _, pod := range pods.Items {
		assetName := pod.Name
		assetId := string(pod.UID)
		assetStartTime := pod.Status.StartTime
		namespace := pod.Namespace
		nodeName := pod.Spec.NodeName
		nodeId, err := getNodeIdFromName(ctx, client, nodeName)
		assetParents := []string{}
		if err == nil {
			nodeAssetName := fmt.Sprintf("%s:%s", "k8s.node", nodeId)
			assetParents = append(assetParents, nodeAssetName)
		}

		log.Info("Publishing pod assets\n")
		internal.Publish(publisher,
			internal.WithAssetTypeAndID("k8s.pod", assetId),
			internal.WithAssetParents(assetParents),
			internal.WithPodData(assetName, assetId, namespace, assetStartTime),
		)
	}

	return nil
}

func getNodeIdFromName(ctx context.Context, client kubernetes.Interface, nodeName string) (string, error) {
	listOptions := metav1.ListOptions{
		FieldSelector: "metadata.name=" + nodeName,
	}
	nodes, err := client.CoreV1().Nodes().List(context.TODO(), listOptions)
	if err != nil {
		return "", err
	}
	for _, node := range nodes.Items {
		return string(node.ObjectMeta.UID), nil
	}
	return "", errors.New("node list is empty for given node name")

}

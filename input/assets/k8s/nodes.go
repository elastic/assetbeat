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
	"time"

	kube "github.com/elastic/elastic-agent-autodiscover/kubernetes"
	"github.com/elastic/inputrunner/input/assets/internal"
	stateless "github.com/elastic/inputrunner/input/v2/input-stateless"

	"github.com/elastic/elastic-agent-libs/logp"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kuberntescli "k8s.io/client-go/kubernetes"
)

type node struct {
	publisher stateless.Publisher
	watcher   kube.Watcher
	client    kuberntescli.Interface
	logger    *logp.Logger
	ctx       context.Context
}

// watchK8sNodes initiates a watcher of kubernetes nodes
func watchK8sNodes(ctx context.Context, log *logp.Logger, client kuberntescli.Interface, publisher stateless.Publisher) error {
	watcher, err := kube.NewNamedWatcher("node", client, &kube.Node{}, kube.WatchOptions{
		SyncTimeout:  10 * time.Minute,
		Node:         "",
		Namespace:    "",
		HonorReSyncs: true,
	}, nil)

	if err != nil {
		log.Errorf("could not create kubernetes watcher %v", err)
		return err
	}

	n := &node{
		publisher: publisher,
		watcher:   watcher,
		client:    client,
		logger:    log,
		ctx:       ctx,
	}

	watcher.AddEventHandler(n)

	log.Infof("start watching for nodes")
	go n.Start()

	return nil
}

// Start starts the eventer
func (n *node) Start() error {
	return n.watcher.Start()
}

// Stop stops the eventer
func (n *node) Stop() {
	n.watcher.Stop()
}

// OnUpdate handles events for pods that have been updated.
func (n *node) OnUpdate(obj interface{}) {
	o := obj.(*kube.Node)
	n.logger.Infof("Watcher Node update: %+v", o.Name)
}

// OnDelete stops pod objects that are deleted.
func (n *node) OnDelete(obj interface{}) {
	o := obj.(*kube.Node)
	n.logger.Infof("Watcher Node delete: %+v", o.Name)

}

// OnAdd ensures processing of node objects that are newly added.
func (n *node) OnAdd(obj interface{}) {
	o := obj.(*kube.Node)
	n.logger.Infof("Watcher Node add: %+v", o.Name)

	assetProviderId := o.Spec.ProviderID
	assetId := string(o.ObjectMeta.UID)
	assetStartTime := o.ObjectMeta.CreationTimestamp
	assetParents := []string{}

	n.logger.Info("Publishing nodes assets\n")
	internal.Publish(n.publisher,
		internal.WithAssetTypeAndID("k8s.node", assetId),
		internal.WithAssetParents(assetParents),
		internal.WithNodeData(o.Name, assetProviderId, &assetStartTime),
	)
}

// getNodeIdFromName returns kubernetes node id from a provided node name
func getNodeIdFromName(ctx context.Context, client kuberntescli.Interface, nodeName string) (string, error) {
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

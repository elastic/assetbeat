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
	"time"

	stateless "github.com/elastic/beats/v7/filebeat/input/v2/input-stateless"
	kube "github.com/elastic/elastic-agent-autodiscover/kubernetes"
	"github.com/elastic/inputrunner/input/assets/internal"

	"github.com/elastic/elastic-agent-libs/logp"

	kuberntescli "k8s.io/client-go/kubernetes"
)

type node struct {
	watcher kube.Watcher
	client  kuberntescli.Interface
	logger  *logp.Logger
	ctx     context.Context
}

// getNodeWatcher initiates and returns a watcher of kubernetes nodes
func getNodeWatcher(ctx context.Context, log *logp.Logger, client kuberntescli.Interface, timeout time.Duration) (kube.Watcher, error) {
	watcher, err := kube.NewNamedWatcher("node", client, &kube.Node{}, kube.WatchOptions{
		SyncTimeout:  timeout,
		Node:         "",
		Namespace:    "",
		HonorReSyncs: true,
	}, nil)

	if err != nil {
		log.Errorf("could not create kubernetes watcher %v", err)
		return nil, err
	}

	n := &node{
		watcher: watcher,
		client:  client,
		logger:  log,
		ctx:     ctx,
	}

	watcher.AddEventHandler(n)

	return watcher, nil
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
	n.logger.Debugf("Watcher Node update: %+v", o.Name)
}

// OnDelete stops pod objects that are deleted.
func (n *node) OnDelete(obj interface{}) {
	o := obj.(*kube.Node)
	n.logger.Debugf("Watcher Node delete: %+v", o.Name)
}

// OnAdd ensures processing of node objects that are newly added.
func (n *node) OnAdd(obj interface{}) {
	o := obj.(*kube.Node)
	n.logger.Debugf("Watcher Node add: %+v", o.Name)
}

// getNodeIdFromName returns kubernetes node id from a provided node name
func getNodeIdFromName(nodeName string, nodeWatcher kube.Watcher) (string, error) {
	node, exists, err := nodeWatcher.Store().GetByKey(nodeName)
	if err != nil {
		return "", err
	}
	if !exists {
		return "", fmt.Errorf("node with name %s does not exist in cache", nodeName)
	}
	n := node.(*kube.Node)
	return string(n.ObjectMeta.UID), nil
}

// publishK8sNodes publishes the node assets stored in node watcher cache
func publishK8sNodes(ctx context.Context, log *logp.Logger, indexNamespace string, publisher stateless.Publisher, watcher kube.Watcher) {
	log.Info("Publishing nodes assets\n")
	assetType := "k8s.node"
	for _, obj := range watcher.Store().List() {
		o, ok := obj.(*kube.Node)
		if ok {
			log.Debug("Publish Node: %+v", o.Name)

			assetProviderId := o.Spec.ProviderID
			assetId := string(o.ObjectMeta.UID)
			assetStartTime := o.ObjectMeta.CreationTimestamp
			assetParents := []string{}

			internal.Publish(publisher,
				internal.WithAssetTypeAndID(assetType, assetId),
				internal.WithAssetParents(assetParents),
				internal.WithNodeData(o.Name, assetProviderId, &assetStartTime),
				internal.WithIndex(assetType, indexNamespace),
			)
		} else {
			log.Error("Publishing nodes assets failed. Type assertion of node object failed")
		}

	}

}

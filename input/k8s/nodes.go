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

	"github.com/elastic/assetbeat/input/internal"
	stateless "github.com/elastic/beats/v7/filebeat/input/v2/input-stateless"
	kube "github.com/elastic/elastic-agent-autodiscover/kubernetes"
	"github.com/elastic/elastic-agent-libs/mapstr"

	"github.com/elastic/elastic-agent-libs/logp"

	kuberntescli "k8s.io/client-go/kubernetes"
)

type node struct {
	watcher kube.Watcher
	client  kuberntescli.Interface
	logger  *logp.Logger
	ctx     context.Context
}

const (
	// metadataHost is the IP that each of the  GCP uses for metadata service.
	metadataHost   = "169.254.169.254"
	gceMetadataURI = "/computeMetadata/v1/?recursive=true&alt=json"
)

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
func publishK8sNodes(ctx context.Context, log *logp.Logger, publisher stateless.Publisher, watcher kube.Watcher, isInCluster bool) {
	log.Info("Publishing nodes assets\n")
	assetType := "k8s.node"
	assetKind := "host"
	var assetParents []string

	// Get the first stored node to extract the cluster uid in case of GCP.
	// Only in case of running InCluster
	log.Debugf("Is in cluster is %s", isInCluster)
	if isInCluster {
		if len(watcher.Store().List()) > 0 {
			if n1, ok := watcher.Store().List()[0].(*kube.Node); ok {
				if getCspFromProviderId(n1.Spec.ProviderID) == "gcp" {
					clusterUid, err := getGKEClusterUid(ctx, log, newhttpFetcher())
					if err != nil {
						log.Debugf("Unable to fetch cluster uid from metadata: %+v \n", err)
					}
					assetParents = append(assetParents, fmt.Sprintf("%s:%s", "cluster", clusterUid))
				}
			}
		}
	}

	for _, obj := range watcher.Store().List() {
		o, ok := obj.(*kube.Node)
		if ok {
			log.Debugf("Publish Node: %+v", o.Name)
			metadata := mapstr.M{
				"state": getNodeState(o),
			}
			log.Debug("Node status: ", metadata["state"])
			instanceId := getInstanceId(o)
			log.Debug("Node instance id: ", instanceId)
			assetId := string(o.ObjectMeta.UID)
			assetStartTime := o.ObjectMeta.CreationTimestamp
			options := []internal.AssetOption{
				internal.WithAssetKindAndID(assetKind, assetId),
				internal.WithAssetType(assetType),
				internal.WithAssetName(o.Name),
				internal.WithAssetMetadata(metadata),
				internal.WithNodeData(o.Name, &assetStartTime),
			}
			if instanceId != "" {
				options = append(options, internal.WithCloudInstanceId(instanceId))
			}
			if assetParents != nil {
				options = append(options, internal.WithAssetParents(assetParents))
			}
			internal.Publish(publisher, nil, options...)

		} else {
			log.Error("Publishing nodes assets failed. Type assertion of node object failed")
		}
	}
}

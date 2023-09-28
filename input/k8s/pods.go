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
	"github.com/elastic/elastic-agent-libs/logp"

	kuberntescli "k8s.io/client-go/kubernetes"
)

type pod struct {
	watcher kube.Watcher
	client  kuberntescli.Interface
	logger  *logp.Logger
	ctx     context.Context
}

// getPodWatcher initiates and returns a watcher of kubernetes pods
func getPodWatcher(ctx context.Context, log *logp.Logger, client kuberntescli.Interface, timeout time.Duration) (kube.Watcher, error) {
	watcher, err := kube.NewNamedWatcher("pod", client, &kube.Pod{}, kube.WatchOptions{
		SyncTimeout:  timeout,
		Node:         "",
		Namespace:    "",
		HonorReSyncs: true,
	}, nil)

	if err != nil {
		log.Errorf("could not create kubernetes watcher %v", err)
		return nil, err
	}

	p := &pod{
		watcher: watcher,
		client:  client,
		logger:  log,
		ctx:     ctx,
	}

	watcher.AddEventHandler(p)

	return watcher, nil
}

// Start starts the eventer
func (p *pod) Start() error {
	return p.watcher.Start()
}

// Stop stops the eventer
func (p *pod) Stop() {
	p.watcher.Stop()
}

// OnUpdate handles events for pods that have been updated.
func (p *pod) OnUpdate(obj interface{}) {
	o := obj.(*kube.Pod)
	p.logger.Debugf("Watcher Pod update: %+v", o.Name)
}

// OnDelete stops pod objects that are deleted.
func (p *pod) OnDelete(obj interface{}) {
	o := obj.(*kube.Pod)
	p.logger.Debugf("Watcher Pod delete: %+v", o.Name)
}

// OnAdd ensures processing of pod objects that are newly added.
func (p *pod) OnAdd(obj interface{}) {
	o := obj.(*kube.Pod)
	p.logger.Debugf("Watcher Pod add: %+v", o.Name)
}

// publishK8sPods publishes the pod assets stored in pod watcher cache
func publishK8sPods(ctx context.Context, log *logp.Logger, publisher stateless.Publisher, podWatcher, nodeWatcher kube.Watcher) {

	log.Info("Publishing pod assets\n")
	assetType := "k8s.pod"
	assetKind := "container_group"
	for _, obj := range podWatcher.Store().List() {
		o, ok := obj.(*kube.Pod)
		if ok {
			log.Debugf("Publish Pod: %+v", o.Name)
			assetName := o.Name
			assetId := string(o.UID)
			assetStartTime := o.Status.StartTime
			namespace := o.Namespace
			nodeName := o.Spec.NodeName

			assetParents := []string{}
			if nodeWatcher != nil {
				nodeId, err := getNodeIdFromName(nodeName, nodeWatcher)
				if err == nil {
					nodeAssetName := fmt.Sprintf("%s:%s", "host", nodeId)
					assetParents = append(assetParents, nodeAssetName)
				} else {
					log.Errorf("pod asset parents not collected: %w", err)
				}
			}

			internal.Publish(publisher, nil,
				internal.WithAssetKindAndID(assetKind, assetId),
				internal.WithAssetType(assetType),
				internal.WithAssetParents(assetParents),
				internal.WithAssetName(assetName),
				internal.WithPodData(assetName, assetId, namespace, assetStartTime),
			)
		} else {
			log.Error("Publishing pod assets failed. Type assertion of pod object failed")
		}

	}
}

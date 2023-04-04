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
	"strings"
	"time"

	kube "github.com/elastic/elastic-agent-autodiscover/kubernetes"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/inputrunner/input/assets/internal"
	stateless "github.com/elastic/inputrunner/input/v2/input-stateless"

	"k8s.io/apimachinery/pkg/api/meta"
	kuberntescli "k8s.io/client-go/kubernetes"
)

type pod struct {
	watcher kube.Watcher
	client  kuberntescli.Interface
	logger  *logp.Logger
	ctx     context.Context
}

// watchK8sPods initiates a watcher of kubernetes pods
func watchK8sPods(ctx context.Context, log *logp.Logger, client kuberntescli.Interface, timeout time.Duration) (kube.Watcher, error) {
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

	log.Infof("start watching for pods")
	go p.Start()

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

	// Get metadata of the object
	accessor, err := meta.Accessor(o)
	if err != nil {
		return
	}
	meta := map[string]string{}
	for _, ref := range accessor.GetOwnerReferences() {
		if ref.Controller != nil && *ref.Controller {
			switch ref.Kind {
			// grow this list as we keep adding more `state_*` metricsets
			case "Deployment",
				"ReplicaSet",
				"StatefulSet",
				"DaemonSet",
				"Job",
				"CronJob":
				meta[strings.ToLower(ref.Kind)+".name"] = ref.Name
			}
		}
	}
}

// OnDelete stops pod objects that are deleted.
func (p *pod) OnDelete(obj interface{}) {
	o := obj.(*kube.Pod)
	p.logger.Debugf("Watcher Pod delete: %+v", o.Name)

	// Get metadata of the object
	accessor, err := meta.Accessor(o)
	if err != nil {
		return
	}
	meta := map[string]string{}
	for _, ref := range accessor.GetOwnerReferences() {
		if ref.Controller != nil && *ref.Controller {
			switch ref.Kind {
			// grow this list as we keep adding more `state_*` metricsets
			case "Deployment",
				"ReplicaSet",
				"StatefulSet",
				"DaemonSet",
				"Job",
				"CronJob":
				meta[strings.ToLower(ref.Kind)+".name"] = ref.Name
			}
		}
	}
}

// OnAdd ensures processing of pod objects that are newly added.
func (p *pod) OnAdd(obj interface{}) {
	o := obj.(*kube.Pod)
	p.logger.Debugf("Watcher Pod add: %+v", o.Name)
}

// publishK8sPods publishes the pod assets stored in pod watcher cache
func publishK8sPods(log *logp.Logger, publisher stateless.Publisher, podWatcher, nodeWatcher kube.Watcher) {

	for _, obj := range podWatcher.Store().List() {
		o := obj.(*kube.Pod)
		log.Debugf("Publish Pod: %+v", o.Name)

		// Get metadata of the object
		accessor, err := meta.Accessor(o)
		if err != nil {
			return
		}
		meta := map[string]string{}
		for _, ref := range accessor.GetOwnerReferences() {
			if ref.Controller != nil && *ref.Controller {
				switch ref.Kind {
				// grow this list as we keep adding more `state_*` metricsets
				case "Deployment",
					"ReplicaSet",
					"StatefulSet",
					"DaemonSet",
					"Job",
					"CronJob":
					meta[strings.ToLower(ref.Kind)+".name"] = ref.Name
				}
			}
		}

		assetName := o.Name
		assetId := string(o.UID)
		assetStartTime := o.Status.StartTime
		namespace := o.Namespace
		nodeName := o.Spec.NodeName

		assetParents := []string{}
		if nodeWatcher != nil {
			nodeId, err := getNodeIdFromName(nodeName, nodeWatcher)
			if err == nil {
				nodeAssetName := fmt.Sprintf("%s:%s", "k8s.node", nodeId)
				assetParents = append(assetParents, nodeAssetName)
			} else {
				log.Errorf("pod asset parents not collected: %w", err)
			}
		}

		log.Info("Publishing pod assets\n")
		internal.Publish(publisher,
			internal.WithAssetTypeAndID("k8s.pod", assetId),
			internal.WithAssetParents(assetParents),
			internal.WithPodData(assetName, assetId, namespace, assetStartTime),
		)
	}

}

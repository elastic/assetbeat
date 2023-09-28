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

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/elastic/assetbeat/input/internal"
	stateless "github.com/elastic/beats/v7/filebeat/input/v2/input-stateless"
	kube "github.com/elastic/elastic-agent-autodiscover/kubernetes"
	"github.com/elastic/elastic-agent-libs/logp"
)

// publishK8sPods publishes the pod assets stored in pod watcher cache
func publishK8sContainers(ctx context.Context, log *logp.Logger, publisher stateless.Publisher, podWatcher kube.Watcher) {
	log.Info("Publishing container assets\n")
	assetType := "k8s.container"
	assetKind := "container"
	for _, obj := range podWatcher.Store().List() {
		o, ok := obj.(*kube.Pod)
		if ok {
			log.Debugf("Publish Pod: %+v", o.Name)
			parentId := string(o.UID)
			parentEan := fmt.Sprintf("%s:%s", "container_group", parentId)
			assetParents := []string{parentEan}
			namespace := o.Namespace

			containers := kube.GetContainersInPod(o)
			for _, c := range containers {
				// If it doesn't have an ID, container doesn't exist in
				// the runtime
				if c.ID == "" {
					continue
				}
				assetId := c.ID
				assetName := c.Spec.Name
				cPhase := c.Status.State
				state := ""
				assetStartTime := metav1.Time{}
				if cPhase.Waiting != nil {
					state = "Waiting"
				} else if cPhase.Running != nil {
					state = "Running"
					assetStartTime = cPhase.Running.StartedAt
				} else if cPhase.Terminated != nil {
					state = "Terminated"
					assetStartTime = cPhase.Terminated.StartedAt
				}

				internal.Publish(publisher, nil,
					internal.WithAssetKindAndID(assetKind, assetId),
					internal.WithAssetType(assetType),
					internal.WithAssetName(assetName),
					internal.WithAssetParents(assetParents),
					internal.WithContainerData(assetName, assetId, namespace, state, &assetStartTime),
				)
			}
		} else {
			log.Error("Publishing pod assets failed. Type assertion of pod object failed")
		}

	}
}

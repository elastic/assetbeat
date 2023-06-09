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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sfake "k8s.io/client-go/kubernetes/fake"

	"github.com/elastic/assetbeat/input/internal"
	"github.com/elastic/assetbeat/input/testutil"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

var startTime = metav1.Time{Time: time.Date(2021, 8, 15, 14, 30, 45, 100, time.Local)}

func TestPublishK8sPodAsset(t *testing.T) {
	for _, tt := range []struct {
		name  string
		event beat.Event

		assetName string
		assetType string
		assetKind string
		assetID   string
		parents   []string
		children  []string
	}{
		{
			name: "publish pod",
			event: beat.Event{
				Fields: mapstr.M{
					"asset.type":                "k8s.pod",
					"asset.kind":                "container_group",
					"asset.id":                  "a375d24b-fa20-4ea6-a0ee-1d38671d2c09",
					"asset.ean":                 "container_group:a375d24b-fa20-4ea6-a0ee-1d38671d2c09",
					"asset.parents":             []string{},
					"kubernetes.pod.name":       "foo",
					"kubernetes.pod.uid":        "a375d24b-fa20-4ea6-a0ee-1d38671d2c09",
					"kubernetes.pod.start_time": &startTime,
					"kubernetes.namespace":      "default",
				},
				Meta: mapstr.M{
					"index": "assets-k8s.pod-default",
				},
			},

			assetName: "foo",
			assetType: "k8s.pod",
			assetKind: "container_group",
			assetID:   "a375d24b-fa20-4ea6-a0ee-1d38671d2c09",
			parents:   []string{},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			publisher := testutil.NewInMemoryPublisher()

			internal.Publish(publisher, nil,
				internal.WithAssetKindAndID(tt.assetKind, tt.assetID),
				internal.WithAssetType(tt.assetType),
				internal.WithAssetParents(tt.parents),
				internal.WithPodData(tt.assetName, tt.assetID, "default", &startTime),
				internal.WithIndex(tt.assetType, ""),
			)
			assert.Equal(t, 1, len(publisher.Events))
			assert.Equal(t, tt.event, publisher.Events[0])
		})
	}
}

func TestPublishK8sNodeAsset(t *testing.T) {
	for _, tt := range []struct {
		name  string
		event beat.Event

		assetName  string
		assetType  string
		assetKind  string
		assetID    string
		instanceID string
		parents    []string
		children   []string
	}{
		{
			name: "publish node",
			event: beat.Event{
				Fields: mapstr.M{
					"asset.type":                 "k8s.node",
					"asset.kind":                 "host",
					"asset.id":                   "60988eed-1885-4b63-9fa4-780206969deb",
					"asset.ean":                  "host:60988eed-1885-4b63-9fa4-780206969deb",
					"asset.parents":              []string{},
					"kubernetes.node.name":       "ip-172-31-29-242.us-east-2.compute.internal",
					"kubernetes.node.start_time": &startTime,
					"cloud.instance.id":          "i-0699b78f46f0fa248",
				},
				Meta: mapstr.M{
					"index": "assets-k8s.node-default",
				},
			},

			assetName:  "ip-172-31-29-242.us-east-2.compute.internal",
			assetType:  "k8s.node",
			assetKind:  "host",
			assetID:    "60988eed-1885-4b63-9fa4-780206969deb",
			instanceID: "i-0699b78f46f0fa248",
			parents:    []string{},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			publisher := testutil.NewInMemoryPublisher()

			internal.Publish(publisher, nil,
				internal.WithAssetKindAndID(tt.assetKind, tt.assetID),
				internal.WithAssetType(tt.assetType),
				internal.WithAssetParents(tt.parents),
				internal.WithNodeData(tt.assetName, &startTime),
				internal.WithIndex(tt.assetType, ""),
				internal.WithCloudInstanceId(tt.instanceID),
			)
			assert.Equal(t, 1, len(publisher.Events))
			assert.Equal(t, tt.event, publisher.Events[0])
		})
	}
}

func TestCollectK8sAssets(t *testing.T) {
	client := k8sfake.NewSimpleClientset()
	log := logp.NewLogger("mylogger")
	podWatcher, err := getPodWatcher(context.Background(), log, client, time.Second*60)
	if err != nil {
		t.Fatalf("error initiating Pod watcher")
	}

	input := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "mypod",
			UID:       "a375d24b-fa20-4ea6-a0ee-1d38671d2c09",
			Namespace: "default",
			Labels: map[string]string{
				"foo": "bar",
			},
			Annotations: map[string]string{
				"app": "production",
			},
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: "v1",
		},
		Spec: v1.PodSpec{
			NodeName: "testnode",
		},
		Status: v1.PodStatus{PodIP: "127.0.0.5"},
	}
	_ = podWatcher.Store().Add(input)

	watchersMap := &watchersMap{}
	watchersMap.watchers.Store("pod", podWatcher)
	publisher := testutil.NewInMemoryPublisher()
	cfg := defaultConfig()
	cfg.AssetTypes = []string{"k8s.pod"}
	collectK8sAssets(context.Background(), log, cfg, publisher, watchersMap)
	time.Sleep(1 * time.Second)
	assert.Equal(t, 1, len(publisher.Events))
}

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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sfake "k8s.io/client-go/kubernetes/fake"

	"github.com/elastic/assetbeat/input/testutil"
	"github.com/elastic/elastic-agent-autodiscover/kubernetes"
	"github.com/elastic/elastic-agent-libs/logp"
)

func TestGetNodeWatcher(t *testing.T) {
	client := k8sfake.NewSimpleClientset()
	log := logp.NewLogger("mylogger")
	_, err := getNodeWatcher(context.Background(), log, client, time.Second*60)
	if err != nil {
		t.Fatalf("error initiating Node watcher")
	}
	assert.NoError(t, err)
}

func TestGetNodeIdFromName(t *testing.T) {
	for _, tt := range []struct {
		name     string
		nodeName string
		input    kubernetes.Resource
		output   error
	}{
		{
			name:     "node not found",
			nodeName: "node2",
			input: &v1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "node1",
					UID:  "60988eed-1885-4b63-9fa4-780206969deb",
					Labels: map[string]string{
						"foo": "bar",
					},
					Annotations: map[string]string{
						"key1": "value1",
						"key2": "value2",
					},
				},
				TypeMeta: metav1.TypeMeta{
					Kind:       "Node",
					APIVersion: "v1",
				},
				Status: v1.NodeStatus{
					Addresses: []v1.NodeAddress{{Type: v1.NodeHostName, Address: "node1"}},
				},
			},
			output: fmt.Errorf("node with name %s does not exist in cache", "node2"),
		},
		{
			name:     "node found",
			nodeName: "node2",
			input: &v1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "node2",
					UID:  "60988eed-1885-4b63-9fa4-780206969deb",
					Labels: map[string]string{
						"foo": "bar",
					},
					Annotations: map[string]string{
						"key1": "value1",
						"key2": "value2",
					},
				},
				TypeMeta: metav1.TypeMeta{
					Kind:       "Node",
					APIVersion: "v1",
				},
				Status: v1.NodeStatus{
					Addresses: []v1.NodeAddress{{Type: v1.NodeHostName, Address: "node2"}},
				},
			},
			output: nil,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			client := k8sfake.NewSimpleClientset()
			log := logp.NewLogger("mylogger")
			nodeWatcher, _ := getNodeWatcher(context.Background(), log, client, time.Second*60)
			_ = nodeWatcher.Store().Add(tt.input)
			_, err := getNodeIdFromName(tt.nodeName, nodeWatcher)
			assert.Equal(t, err, tt.output)
		})
	}
}

func TestPublishK8sNodes(t *testing.T) {
	client := k8sfake.NewSimpleClientset()
	log := logp.NewLogger("mylogger")
	nodeWatcher, err := getNodeWatcher(context.Background(), log, client, time.Second*60)
	if err != nil {
		t.Fatalf("error initiating Node watcher")
	}
	input := &v1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: "nodeName",
			UID:  "60988eed-1885-4b63-9fa4-780206969deb",
			Labels: map[string]string{
				"foo": "bar",
			},
			Annotations: map[string]string{
				"key": "value",
			},
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "Node",
			APIVersion: "v1",
		},
		Status: v1.NodeStatus{
			Addresses: []v1.NodeAddress{{Type: v1.NodeHostName, Address: "node1"}},
		},
	}
	_ = nodeWatcher.Store().Add(input)
	publisher := testutil.NewInMemoryPublisher()
	publishK8sNodes(context.Background(), log, "", publisher, nodeWatcher, false)

	assert.Equal(t, 1, len(publisher.Events))
}

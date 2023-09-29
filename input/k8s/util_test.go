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
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/elastic/elastic-agent-autodiscover/kubernetes"
	kube "github.com/elastic/elastic-agent-autodiscover/kubernetes"
	"github.com/elastic/elastic-agent-libs/logp"
)

type mockHttpResponse struct {
	response []byte
}

func newMockhttpResponse(response []byte) mockHttpResponse {
	return mockHttpResponse{response: response}
}
func (c mockHttpResponse) FetchResponse(ctx context.Context, url string, headers map[string]string) ([]byte, error) {
	return c.response, nil
}
func TestGetInstanceId(t *testing.T) {
	for _, tt := range []struct {
		name   string
		input  kubernetes.Resource
		output string
	}{
		{
			name: "AWS node",
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
				Spec: v1.NodeSpec{
					ProviderID: "aws:///us-east-2b/i-0699b78f46f0fa248",
				},
			},
			output: "i-0699b78f46f0fa248",
		},
		{
			name: "AWS node Fargate",
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
				Spec: v1.NodeSpec{
					ProviderID: "aws:///us-east-2c/fa80a30ea9-6a2c0e0c771e4e8caa80f702f9821271/fargate-ip-192-168-104-15.us-east-2.compute.internal",
				},
			},
			output: "",
		},
		{
			name: "GCP node",
			input: &v1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "node1",
					UID:  "60988eed-1885-4b63-9fa4-780206969deb",
					Labels: map[string]string{
						"foo": "bar",
					},
					Annotations: map[string]string{
						"key1":                                 "value1",
						"key2":                                 "value2",
						"container.googleapis.com/instance_id": "5445971517456914360",
					},
				},
				TypeMeta: metav1.TypeMeta{
					Kind:       "Node",
					APIVersion: "v1",
				},
				Status: v1.NodeStatus{
					Addresses: []v1.NodeAddress{{Type: v1.NodeHostName, Address: "node1"}},
				},
				Spec: v1.NodeSpec{
					ProviderID: "gce://elastic-observability/us-central1-c/gke-michaliskatsoulis-te-default-pool-41126842-55kg",
				},
			},
			output: "5445971517456914360",
		},
		{
			name: "Azure Node",
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
					NodeInfo:  v1.NodeSystemInfo{SystemUUID: "ebf2fd19-ee80-4751-91e5-087c4fe39845"},
				},
				Spec: v1.NodeSpec{
					ProviderID: "azure:///subscriptions/11232-12321-sdsads-123/resourceGroups/mc_michaliskatsoulisgroup_katsoulistest2_eastus/providers/Microsoft.Compute/virtualMachineScaleSets/aks-agentpool-56241798-vmss/virtualMachines/0",
				},
			},
			output: "ebf2fd19-ee80-4751-91e5-087c4fe39845",
		},
		{
			name: "No CSP Node (kind)",
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
				Spec: v1.NodeSpec{
					ProviderID: "kind://docker/kind/kind-worker",
				},
			},
			output: "",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			n := tt.input.(*kube.Node)
			providerId := getInstanceId(n)
			assert.Equal(t, providerId, tt.output)
		})
	}
}

func TestGetCspFromProviderId(t *testing.T) {
	for _, tt := range []struct {
		name   string
		input  string
		output string
	}{
		{
			name:   "AWS node",
			input:  "aws:///us-east-2b/i-0699b78f46f0fa248",
			output: "aws",
		},
		{
			name:   "GCP node",
			input:  "gce://elastic-observability/us-central1-c/gke-michaliskatsoulis-te-default-pool-41126842-55kg",
			output: "gcp",
		},
		{
			name:   "No CSP Node (kind)",
			input:  "kind://docker/kind/kind-worker",
			output: "",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			csp := getCspFromProviderId(tt.input)
			assert.Equal(t, csp, tt.output)
		})
	}
}

func TestGetGKEClusterUid(t *testing.T) {
	for _, tt := range []struct {
		name   string
		input  map[string]interface{}
		output string
	}{
		{
			name: "Normal case, cluster_uid present",
			input: map[string]interface{}{
				"instance": map[string]interface{}{
					"attributes": map[string]interface{}{
						"cluster-uid": "ed436d761637404fa772b2822ab1036f14cf4727c4e54a28ac86f5ab8dcda4af",
					},
				},
			},
			output: "ed436d761637404fa772b2822ab1036f14cf4727c4e54a28ac86f5ab8dcda4af",
		},
		{
			name: "cluster_uid present but empty",
			input: map[string]interface{}{
				"instance": map[string]interface{}{
					"attributes": map[string]interface{}{
						"cluster-uid": "",
					},
				},
			},
			output: "",
		},
		{
			name:   "cluster_uid not present",
			input:  map[string]interface{}{},
			output: "",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			log := logp.NewLogger("mylogger")
			ctx := context.Background()
			reponse, _ := json.Marshal(tt.input)
			hf := newMockhttpResponse(reponse)
			cuid, _ := getGKEClusterUid(ctx, log, hf)
			assert.Equal(t, cuid, tt.output)
		})
	}
}

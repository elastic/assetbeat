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

package gcp

import (
	"context"
	"fmt"
	"github.com/elastic/assetbeat/input/internal"
	"regexp"
	"testing"

	"github.com/elastic/go-freelru"

	"github.com/gogo/protobuf/proto"

	"github.com/elastic/assetbeat/input/testutil"
	"github.com/elastic/beats/v7/libbeat/beat"

	"github.com/googleapis/gax-go/v2"

	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"

	compute "cloud.google.com/go/compute/apiv1"
	"cloud.google.com/go/compute/apiv1/computepb"
	"cloud.google.com/go/container/apiv1/containerpb"
	"github.com/stretchr/testify/assert"
)

type ClustersClientStub struct {
	Clusters map[string]*containerpb.ListClustersResponse
}

func (s *ClustersClientStub) ListClusters(ctx context.Context, req *containerpb.ListClustersRequest, opts ...gax.CallOption) (*containerpb.ListClustersResponse, error) {
	m := findGKEProjectRe.FindStringSubmatch(req.Parent)
	fmt.Println(req.Parent)
	project := m[1]
	fmt.Println(project)
	return s.Clusters[project], nil
}

var findGKEProjectRe = regexp.MustCompile("projects/([a-z_-]+)/locations/([0-9a-z_,-]+)")

func TestCollectGKEAssets(t *testing.T) {
	vpcAssetsCache := getTestVpcCache()
	var children []string
	var parents []string
	for _, tt := range []struct {
		name string

		ctx                context.Context
		cfg                config
		apiResponses       map[string]*containerpb.ListClustersResponse
		instances          map[string]*StubAggregatedInstanceListIterator
		computeAssetsCache *freelru.LRU[string, *computeInstance]
		expectedEvents     []beat.Event
	}{
		{
			name: "with no project specified",

			ctx: context.Background(),
			cfg: config{},
		},
		{
			name: "with one project specified",

			ctx: context.Background(),
			cfg: config{
				Projects: []string{"my_project"},
			},

			apiResponses: map[string]*containerpb.ListClustersResponse{
				"my_project": {
					Clusters: []*containerpb.Cluster{
						{
							Id:       "1",
							Location: "europe-west1",
							Network:  "my_network",
							NetworkConfig: &containerpb.NetworkConfig{
								Network: "projects/my_project/global/networks/my_network",
							},
							Status: containerpb.Cluster_RUNNING,
							Name:   "myCluster",
						},
					},
				},
			},
			instances: map[string]*StubAggregatedInstanceListIterator{
				"my_project": {
					ReturnScopedInstancesList: []compute.InstancesScopedListPair{{
						Key:   "europe-west-1",
						Value: &computepb.InstancesScopedList{},
					},
					},
				},
			},
			computeAssetsCache: getTestComputeCache(),
			expectedEvents: []beat.Event{
				{
					Fields: mapstr.M{
						"asset.ean":            "cluster:1",
						"asset.id":             "1",
						"asset.type":           "k8s.cluster",
						"asset.kind":           "cluster",
						"asset.parents":        []string{"network:1"},
						"asset.metadata.state": "RUNNING",
						"asset.name":           "myCluster",
						"asset.children":       children,
						"cloud.account.id":     "my_project",
						"cloud.provider":       "gcp",
						"cloud.region":         "europe-west1",
					},
					Meta: mapstr.M{
						"index": internal.GetDefaultIndexName(),
					},
				},
			},
		},
		{
			name: "with multiple projects specified",

			ctx: context.Background(),
			cfg: config{
				Projects: []string{"my_project", "my_second_project"},
			},

			apiResponses: map[string]*containerpb.ListClustersResponse{
				"my_project": {
					Clusters: []*containerpb.Cluster{
						{
							Id:       "1",
							Location: "europe-west1",
							Network:  "my_network",
							NetworkConfig: &containerpb.NetworkConfig{
								Network: "projects/my_project/global/networks/my_network",
							},
							Status: containerpb.Cluster_RUNNING,
							Name:   "myCluster1",
						},
					},
				},
				"my_second_project": {
					Clusters: []*containerpb.Cluster{
						{
							Id:       "42",
							Location: "us-central",
							Network:  "",
							NetworkConfig: &containerpb.NetworkConfig{
								Network: "",
							},
							Status: containerpb.Cluster_STOPPING,
							Name:   "myCluster2",
						},
					},
				},
			},
			instances: map[string]*StubAggregatedInstanceListIterator{
				"my_project": {
					ReturnScopedInstancesList: []compute.InstancesScopedListPair{{
						Key:   "europe-west-1",
						Value: &computepb.InstancesScopedList{},
					},
					},
				},
				"my_second_project": {
					ReturnScopedInstancesList: []compute.InstancesScopedListPair{{
						Key:   "europe-west-1",
						Value: &computepb.InstancesScopedList{},
					},
					},
				},
			},
			computeAssetsCache: getTestComputeCache(),
			expectedEvents: []beat.Event{
				{
					Fields: mapstr.M{
						"asset.ean":            "cluster:1",
						"asset.id":             "1",
						"asset.type":           "k8s.cluster",
						"asset.kind":           "cluster",
						"asset.parents":        []string{"network:1"},
						"asset.metadata.state": "RUNNING",
						"asset.name":           "myCluster1",
						"asset.children":       children,
						"cloud.account.id":     "my_project",
						"cloud.provider":       "gcp",
						"cloud.region":         "europe-west1",
					},
					Meta: mapstr.M{
						"index": internal.GetDefaultIndexName(),
					},
				},
				{
					Fields: mapstr.M{
						"asset.ean":            "cluster:42",
						"asset.id":             "42",
						"asset.type":           "k8s.cluster",
						"asset.kind":           "cluster",
						"asset.parents":        parents,
						"asset.metadata.state": "STOPPING",
						"asset.name":           "myCluster2",
						"asset.children":       children,
						"cloud.account.id":     "my_second_project",
						"cloud.provider":       "gcp",
						"cloud.region":         "us-central",
					},
					Meta: mapstr.M{
						"index": internal.GetDefaultIndexName(),
					},
				},
			},
		},
		{
			name: "with a regions filter",

			ctx: context.Background(),
			cfg: config{
				Projects: []string{"my_project"},
				Regions:  []string{"us-west1"},
			},

			apiResponses: map[string]*containerpb.ListClustersResponse{
				"my_project": {
					Clusters: []*containerpb.Cluster{
						{
							Id:       "2",
							Location: "us-west1",
							Network:  "my_network",
							NetworkConfig: &containerpb.NetworkConfig{
								Network: "projects/my_project/global/networks/my_network",
							},
							Status: containerpb.Cluster_RUNNING,
							Name:   "myCluster",
						},
					},
				},
			},
			instances: map[string]*StubAggregatedInstanceListIterator{
				"my_project": {
					ReturnScopedInstancesList: []compute.InstancesScopedListPair{{
						Key:   "europe-west-1",
						Value: &computepb.InstancesScopedList{},
					},
					},
				},
			},
			computeAssetsCache: getTestComputeCache(),
			expectedEvents: []beat.Event{
				{
					Fields: mapstr.M{
						"asset.ean":            "cluster:2",
						"asset.id":             "2",
						"asset.type":           "k8s.cluster",
						"asset.kind":           "cluster",
						"asset.parents":        []string{"network:1"},
						"asset.metadata.state": "RUNNING",
						"asset.name":           "myCluster",
						"asset.children":       children,
						"cloud.account.id":     "my_project",
						"cloud.provider":       "gcp",
						"cloud.region":         "us-west1",
					},
					Meta: mapstr.M{
						"index": internal.GetDefaultIndexName(),
					},
				},
			},
		},
		{
			name: "with multiple regions filters",

			ctx: context.Background(),
			cfg: config{
				Projects: []string{"my_project"},
				Regions:  []string{"us-west1,europe-west1"},
			},

			apiResponses: map[string]*containerpb.ListClustersResponse{
				"my_project": {
					Clusters: []*containerpb.Cluster{
						{
							Id:       "2",
							Location: "us-west1",
							Network:  "my_network",
							NetworkConfig: &containerpb.NetworkConfig{
								Network: "projects/my_project/global/networks/my_network",
							},
							Status: containerpb.Cluster_RUNNING,
							Name:   "myCluster1",
						},
						{
							Id:       "1",
							Location: "europe-west1",
							Network:  "my_network",
							NetworkConfig: &containerpb.NetworkConfig{
								Network: "projects/my_project/global/networks/my_network",
							},
							Status: containerpb.Cluster_RUNNING,
							Name:   "myCluster2",
						},
					},
				},
			},
			instances: map[string]*StubAggregatedInstanceListIterator{
				"my_project": {
					ReturnScopedInstancesList: []compute.InstancesScopedListPair{{
						Key:   "europe-west-1",
						Value: &computepb.InstancesScopedList{},
					},
					},
				},
			},
			computeAssetsCache: getTestComputeCache(),
			expectedEvents: []beat.Event{
				{
					Fields: mapstr.M{
						"asset.ean":            "cluster:2",
						"asset.id":             "2",
						"asset.type":           "k8s.cluster",
						"asset.kind":           "cluster",
						"asset.parents":        []string{"network:1"},
						"asset.metadata.state": "RUNNING",
						"asset.name":           "myCluster1",
						"asset.children":       children,
						"cloud.account.id":     "my_project",
						"cloud.provider":       "gcp",
						"cloud.region":         "us-west1",
					},
					Meta: mapstr.M{
						"index": internal.GetDefaultIndexName(),
					},
				},
				{
					Fields: mapstr.M{
						"asset.ean":            "cluster:1",
						"asset.id":             "1",
						"asset.type":           "k8s.cluster",
						"asset.kind":           "cluster",
						"asset.parents":        []string{"network:1"},
						"asset.metadata.state": "RUNNING",
						"asset.name":           "myCluster2",
						"asset.children":       children,
						"cloud.account.id":     "my_project",
						"cloud.provider":       "gcp",
						"cloud.region":         "europe-west1",
					},
					Meta: mapstr.M{
						"index": internal.GetDefaultIndexName(),
					},
				},
			},
		},
		{
			name: "with one project specified and children",

			ctx: context.Background(),
			cfg: config{
				Projects: []string{"my_project"},
			},

			apiResponses: map[string]*containerpb.ListClustersResponse{
				"my_project": {
					Clusters: []*containerpb.Cluster{
						{
							Id:       "1",
							Location: "europe-west1",
							Network:  "my_network",
							NetworkConfig: &containerpb.NetworkConfig{
								Network: "projects/my_project/global/networks/my_network",
							},
							Status: containerpb.Cluster_RUNNING,
							NodePools: []*containerpb.NodePool{
								{
									Name: "mynodepool",
								},
							},
							Name: "myCluster",
						},
					},
				},
			},
			instances: map[string]*StubAggregatedInstanceListIterator{
				"my_project": {
					ReturnScopedInstancesList: []compute.InstancesScopedListPair{{
						Key: "europe-west-1",
						Value: &computepb.InstancesScopedList{
							Instances: []*computepb.Instance{
								{
									Id:   proto.Uint64(123),
									Zone: proto.String("https://www.googleapis.com/compute/v1/projects/my_project/zones/europe-west1-d"),
									NetworkInterfaces: []*computepb.NetworkInterface{
										{
											Network: proto.String("https://www.googleapis.com/compute/v1/projects/my_project/global/networks/my_network"),
										},
									},
									Status: proto.String("RUNNING"),
									Metadata: &computepb.Metadata{
										Items: []*computepb.Items{
											{
												Key:   proto.String("kube-labels"),
												Value: proto.String("cloud.google.com/gke-nodepool=mynodepool"),
											},
										},
									}},
							},
						},
					},
					},
				},
			},
			computeAssetsCache: getTestComputeCache(),
			expectedEvents: []beat.Event{
				{
					Fields: mapstr.M{
						"asset.ean":            "cluster:1",
						"asset.id":             "1",
						"asset.type":           "k8s.cluster",
						"asset.kind":           "cluster",
						"asset.parents":        []string{"network:1"},
						"asset.metadata.state": "RUNNING",
						"asset.children":       []string{"host:123"},
						"asset.name":           "myCluster",
						"cloud.account.id":     "my_project",
						"cloud.provider":       "gcp",
						"cloud.region":         "europe-west1",
					},
					Meta: mapstr.M{
						"index": internal.GetDefaultIndexName(),
					},
				},
			},
		},
		{
			name: "with one project specified and children with empty computeAssetsCache. Get from Api",

			ctx: context.Background(),
			cfg: config{
				Projects: []string{"my_project"},
			},

			apiResponses: map[string]*containerpb.ListClustersResponse{
				"my_project": {
					Clusters: []*containerpb.Cluster{
						{
							Id:       "1",
							Location: "europe-west1",
							Network:  "my_network",
							NetworkConfig: &containerpb.NetworkConfig{
								Network: "projects/my_project/global/networks/my_network",
							},
							Status: containerpb.Cluster_RUNNING,
							NodePools: []*containerpb.NodePool{
								{
									Name: "mynodepool",
								},
							},
							Name: "myCluster",
						},
					},
				},
			},
			instances: map[string]*StubAggregatedInstanceListIterator{
				"my_project": {
					ReturnScopedInstancesList: []compute.InstancesScopedListPair{{
						Key: "europe-west-1",
						Value: &computepb.InstancesScopedList{
							Instances: []*computepb.Instance{
								{
									Id:   proto.Uint64(124),
									Zone: proto.String("https://www.googleapis.com/compute/v1/projects/my_project/zones/europe-west1-d"),
									NetworkInterfaces: []*computepb.NetworkInterface{
										{
											Network: proto.String("https://www.googleapis.com/compute/v1/projects/my_project/global/networks/my_network"),
										},
									},
									Status: proto.String("RUNNING"),
									Metadata: &computepb.Metadata{
										Items: []*computepb.Items{
											{
												Key:   proto.String("kube-labels"),
												Value: proto.String("cloud.google.com/gke-nodepool=mynodepool"),
											},
										},
									}},
							},
						},
					},
					},
				},
			},
			computeAssetsCache: getComputeCache(),
			expectedEvents: []beat.Event{
				{
					Fields: mapstr.M{
						"asset.ean":            "cluster:1",
						"asset.id":             "1",
						"asset.type":           "k8s.cluster",
						"asset.kind":           "cluster",
						"asset.parents":        []string{"network:1"},
						"asset.metadata.state": "RUNNING",
						"asset.children":       []string{"host:124"},
						"asset.name":           "myCluster",
						"cloud.account.id":     "my_project",
						"cloud.provider":       "gcp",
						"cloud.region":         "europe-west1",
					},
					Meta: mapstr.M{
						"index": internal.GetDefaultIndexName(),
					},
				},
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {

			listClusterClient := ClustersClientStub{Clusters: tt.apiResponses}
			listInstanceClient := InstancesClientStub{AggregatedInstanceListIterator: tt.instances}
			listInstanceClientCreator := listInstanceAPIClient{
				AggregatedList: func(ctx context.Context, req *computepb.AggregatedListInstancesRequest, opts ...gax.CallOption) AggregatedInstanceIterator {
					return listInstanceClient.AggregatedList(ctx, req, opts...)
				},
			}
			publisher := testutil.NewInMemoryPublisher()
			log := logp.NewLogger("mylogger")
			err := collectGKEAssets(tt.ctx, tt.cfg, vpcAssetsCache, tt.computeAssetsCache, log, listInstanceClientCreator, &listClusterClient, publisher)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedEvents, publisher.Events)
		})
	}
}

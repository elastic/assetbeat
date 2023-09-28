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
	"github.com/elastic/assetbeat/input/internal"
	"testing"

	compute "cloud.google.com/go/compute/apiv1"
	"cloud.google.com/go/compute/apiv1/computepb"
	"github.com/gogo/protobuf/proto"
	"github.com/googleapis/gax-go/v2"
	"google.golang.org/api/iterator"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/assetbeat/input/testutil"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

type StubAggregatedInstanceListIterator struct {
	iterCounter               int
	ReturnScopedInstancesList []compute.InstancesScopedListPair
	ReturnInstancesError      error
}

func (it *StubAggregatedInstanceListIterator) Next() (compute.InstancesScopedListPair, error) {

	if it.ReturnInstancesError != nil {
		return compute.InstancesScopedListPair{}, it.ReturnInstancesError
	}

	if it.iterCounter == len(it.ReturnScopedInstancesList) {
		return compute.InstancesScopedListPair{}, iterator.Done
	}

	instances := it.ReturnScopedInstancesList[it.iterCounter]
	it.iterCounter++

	return instances, nil

}

type InstancesClientStub struct {
	AggregatedInstanceListIterator map[string]*StubAggregatedInstanceListIterator
}

func (s *InstancesClientStub) AggregatedList(ctx context.Context, req *computepb.AggregatedListInstancesRequest, opts ...gax.CallOption) AggregatedInstanceIterator {
	project := req.Project
	return s.AggregatedInstanceListIterator[project]
}

func TestGetAllComputeInstances(t *testing.T) {
	subnetAssetsCache := getTestSubnetCache()
	var parents []string
	for _, tt := range []struct {
		name string

		ctx            context.Context
		cfg            config
		instances      map[string]*StubAggregatedInstanceListIterator
		expectedEvents []beat.Event
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

			instances: map[string]*StubAggregatedInstanceListIterator{
				"my_project": {
					ReturnScopedInstancesList: []compute.InstancesScopedListPair{{
						Key: "europe-west1-d",
						Value: &computepb.InstancesScopedList{
							Instances: []*computepb.Instance{
								{
									Id:       proto.Uint64(1),
									SelfLink: proto.String("https://www.googleapis.com/compute/v1/projects/elastic-observability/zones/europe-west1-d/instances/my-instance-1"),
									Zone:     proto.String("https://www.googleapis.com/compute/v1/projects/my_project/zones/europe-west1-d"),
									NetworkInterfaces: []*computepb.NetworkInterface{
										{
											Network:    proto.String("https://www.googleapis.com/compute/v1/projects/my_project/global/networks/my_network"),
											Subnetwork: proto.String("https://www.googleapis.com/compute/v1/projects/elastic-observability/regions/us-central1/subnetworks/my_subnet"),
										},
									},
									Status: proto.String("RUNNING"),
									Name:   proto.String("myInstance")},
							},
						},
					},
					},
				},
			},
			expectedEvents: []beat.Event{
				{
					Fields: mapstr.M{
						"asset.ean":            "host:1",
						"asset.id":             "1",
						"asset.type":           "gcp.compute.instance",
						"asset.kind":           "host",
						"asset.parents":        []string{"network:2"},
						"asset.metadata.state": "RUNNING",
						"asset.name":           "myInstance",
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

			instances: map[string]*StubAggregatedInstanceListIterator{
				"my_project": {
					ReturnScopedInstancesList: []compute.InstancesScopedListPair{{
						Key: "europe-west1-d",
						Value: &computepb.InstancesScopedList{
							Instances: []*computepb.Instance{
								{
									Id:       proto.Uint64(1),
									SelfLink: proto.String("https://www.googleapis.com/compute/v1/projects/elastic-observability/zones/europe-west1-d/instances/my-instance-1"),
									Zone:     proto.String("https://www.googleapis.com/compute/v1/projects/my_project/zones/europe-west1-d"),
									Status:   proto.String("PROVISIONING"),
									Name:     proto.String("myInstance1")},
							},
						},
					},
					},
				},
				"my_second_project": {
					ReturnScopedInstancesList: []compute.InstancesScopedListPair{{
						Key: "europe-west1-d",
						Value: &computepb.InstancesScopedList{
							Instances: []*computepb.Instance{
								{
									Id:       proto.Uint64(42),
									SelfLink: proto.String("https://www.googleapis.com/compute/v1/projects/elastic-observability/zones/europe-west1-d/instances/my-instance-2"),
									Zone:     proto.String("https://www.googleapis.com/compute/v1/projects/my_project/zones/europe-west1-d"),
									Status:   proto.String("STOPPED"),
									Name:     proto.String("myInstance2")},
							},
						},
					},
					},
				},
			},
			expectedEvents: []beat.Event{
				{
					Fields: mapstr.M{
						"asset.ean":            "host:1",
						"asset.id":             "1",
						"asset.type":           "gcp.compute.instance",
						"asset.kind":           "host",
						"asset.parents":        parents,
						"asset.metadata.state": "PROVISIONING",
						"asset.name":           "myInstance1",
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
						"asset.ean":            "host:42",
						"asset.id":             "42",
						"asset.type":           "gcp.compute.instance",
						"asset.kind":           "host",
						"asset.parents":        parents,
						"asset.metadata.state": "STOPPED",
						"asset.name":           "myInstance2",
						"cloud.account.id":     "my_second_project",
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
			name: "with a region filter",

			ctx: context.Background(),
			cfg: config{
				Projects: []string{"my_project"},
				Regions:  []string{"us-west1"},
			},

			instances: map[string]*StubAggregatedInstanceListIterator{
				"my_project": {
					ReturnScopedInstancesList: []compute.InstancesScopedListPair{
						{
							Key: "europe-west1-d",
							Value: &computepb.InstancesScopedList{
								Instances: []*computepb.Instance{
									{
										Id:       proto.Uint64(1),
										SelfLink: proto.String("https://www.googleapis.com/compute/v1/projects/elastic-observability/zones/europe-west1-d/instances/my-instance-1"),
										Zone:     proto.String("https://www.googleapis.com/compute/v1/projects/my_project/zones/europe-west1-d"),
										NetworkInterfaces: []*computepb.NetworkInterface{
											{
												Network:    proto.String("https://www.googleapis.com/compute/v1/projects/my_project/global/networks/my_network"),
												Subnetwork: proto.String("https://www.googleapis.com/compute/v1/projects/elastic-observability/regions/us-central1/subnetworks/my_subnet"),
											},
										},
										Status: proto.String("RUNNING"),
										Name:   proto.String("myInstance1")},
								},
							},
						},
						{
							Key: "us-west1-b",
							Value: &computepb.InstancesScopedList{
								Instances: []*computepb.Instance{
									{
										Id:       proto.Uint64(42),
										SelfLink: proto.String("https://www.googleapis.com/compute/v1/projects/elastic-observability/zones/europe-west1-b/instances/my-instance-2"),
										Zone:     proto.String("https://www.googleapis.com/compute/v1/projects/my_project/zones/us-west1-b"),
										Status:   proto.String("RUNNING"),
										Name:     proto.String("myInstance2")},
								},
							},
						},
					},
				},
			},
			expectedEvents: []beat.Event{
				{
					Fields: mapstr.M{
						"asset.ean":            "host:42",
						"asset.id":             "42",
						"asset.type":           "gcp.compute.instance",
						"asset.kind":           "host",
						"asset.parents":        parents,
						"asset.metadata.state": "RUNNING",
						"asset.name":           "myInstance2",
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
	} {
		t.Run(tt.name, func(t *testing.T) {
			log := logp.NewLogger("mylogger")
			publisher := testutil.NewInMemoryPublisher()
			client := InstancesClientStub{AggregatedInstanceListIterator: tt.instances}
			clientCreator := listInstanceAPIClient{
				AggregatedList: func(ctx context.Context, req *computepb.AggregatedListInstancesRequest, opts ...gax.CallOption) AggregatedInstanceIterator {
					return client.AggregatedList(ctx, req, opts...)
				},
			}
			computeAssetsCache := getComputeCache()
			err := collectComputeAssets(tt.ctx, tt.cfg, subnetAssetsCache, computeAssetsCache, clientCreator, publisher, log)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedEvents, publisher.Events)
		})
	}
}

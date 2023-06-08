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
	compute "cloud.google.com/go/compute/apiv1"
	"cloud.google.com/go/compute/apiv1/computepb"
	"context"
	"github.com/gogo/protobuf/proto"
	"github.com/googleapis/gax-go/v2"
	"google.golang.org/api/iterator"
	"testing"

	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/stretchr/testify/assert"
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
	for _, tt := range []struct {
		name string

		ctx               context.Context
		cfg               config
		instances         map[string]*StubAggregatedInstanceListIterator
		expectedInstances []computeInstance
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
						Key: "europe-west-1",
						Value: &computepb.InstancesScopedList{
							Instances: []*computepb.Instance{
								{
									Id:   proto.Uint64(1),
									Zone: proto.String("https://www.googleapis.com/compute/v1/projects/my_project/zones/europe-west1-d"),
									NetworkInterfaces: []*computepb.NetworkInterface{
										{
											Network: proto.String("https://www.googleapis.com/compute/v1/projects/my_project/global/networks/my_network"),
										},
									},
									Status: proto.String("RUNNING")},
							},
						},
					},
					},
				},
			},

			expectedInstances: []computeInstance{
				{
					ID:      "1",
					Region:  "europe-west1",
					Account: "my_project",
					VPCs:    []string{"my_network"},
					Metadata: mapstr.M{
						"state": "RUNNING",
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
						Key: "europe-west-1",
						Value: &computepb.InstancesScopedList{
							Instances: []*computepb.Instance{
								{
									Id:     proto.Uint64(1),
									Zone:   proto.String("https://www.googleapis.com/compute/v1/projects/my_project/zones/europe-west1-d"),
									Status: proto.String("PROVISIONING")},
							},
						},
					},
					},
				},
				"my_second_project": {
					ReturnScopedInstancesList: []compute.InstancesScopedListPair{{
						Key: "europe-west-1",
						Value: &computepb.InstancesScopedList{
							Instances: []*computepb.Instance{
								{
									Id:     proto.Uint64(42),
									Zone:   proto.String("https://www.googleapis.com/compute/v1/projects/my_project/zones/europe-west1-d"),
									Status: proto.String("STOPPED")},
							},
						},
					},
					},
				},
			},

			expectedInstances: []computeInstance{
				{
					ID:      "1",
					Region:  "europe-west1",
					Account: "my_project",
					Metadata: mapstr.M{
						"state": "PROVISIONING",
					},
				},
				{
					ID:      "42",
					Region:  "europe-west1",
					Account: "my_second_project",
					Metadata: mapstr.M{
						"state": "STOPPED",
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
							Key: "europe-west-1",
							Value: &computepb.InstancesScopedList{
								Instances: []*computepb.Instance{
									{
										Id:   proto.Uint64(1),
										Zone: proto.String("https://www.googleapis.com/compute/v1/projects/my_project/zones/europe-west1-d"),
										NetworkInterfaces: []*computepb.NetworkInterface{
											{
												Network: proto.String("https://www.googleapis.com/compute/v1/projects/my_project/global/networks/my_network"),
											},
										},
										Status: proto.String("RUNNING")},
								},
							},
						},
						{
							Key: "us-west1-b",
							Value: &computepb.InstancesScopedList{
								Instances: []*computepb.Instance{
									{
										Id:     proto.Uint64(42),
										Zone:   proto.String("https://www.googleapis.com/compute/v1/projects/my_project/zones/us-west1-b"),
										Status: proto.String("RUNNING")},
								},
							},
						},
					},
				},
			},

			expectedInstances: []computeInstance{
				{
					ID:      "42",
					Region:  "us-west1",
					Account: "my_project",
					Metadata: mapstr.M{
						"state": "RUNNING",
					},
				},
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			client := InstancesClientStub{AggregatedInstanceListIterator: tt.instances}
			clientCreator := listInstanceAPIClient{
				AggregatedList: func(ctx context.Context, req *computepb.AggregatedListInstancesRequest, opts ...gax.CallOption) AggregatedInstanceIterator {
					return client.AggregatedList(ctx, req, opts...)
				},
			}
			instances, err := getAllComputeInstances(tt.ctx, tt.cfg, clientCreator)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedInstances, instances)
		})
	}
}

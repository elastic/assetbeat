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
	"testing"

	"cloud.google.com/go/compute/apiv1/computepb"
	"github.com/gogo/protobuf/proto"
	"github.com/googleapis/gax-go/v2"
	"github.com/stretchr/testify/assert"
	"google.golang.org/api/iterator"

	"github.com/elastic/assetbeat/input/testutil"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

type StubNetworksListIterator struct {
	iterCounter          int
	ReturnNetworksList   []*computepb.Network
	ReturnInstancesError error
}

func (it *StubNetworksListIterator) Next() (*computepb.Network, error) {

	if it.ReturnInstancesError != nil {
		return &computepb.Network{}, it.ReturnInstancesError
	}

	if it.iterCounter == len(it.ReturnNetworksList) {
		return &computepb.Network{}, iterator.Done
	}

	networks := it.ReturnNetworksList[it.iterCounter]
	it.iterCounter++

	return networks, nil

}

type NetworkClientStub struct {
	NetworkListIterator map[string]*StubNetworksListIterator
}

func (s *NetworkClientStub) List(ctx context.Context, req *computepb.ListNetworksRequest, opts ...gax.CallOption) NetworkIterator {
	project := req.Project
	return s.NetworkListIterator[project]
}

type StubSubnetsListIterator struct {
	iterCounter        int
	ReturnSubnetsList  []*computepb.Subnetwork
	ReturnSubnetsError error
}

func (it *StubSubnetsListIterator) Next() (*computepb.Subnetwork, error) {

	if it.ReturnSubnetsError != nil {
		return &computepb.Subnetwork{}, it.ReturnSubnetsError
	}

	if it.iterCounter == len(it.ReturnSubnetsList) {
		return &computepb.Subnetwork{}, iterator.Done
	}

	networks := it.ReturnSubnetsList[it.iterCounter]
	it.iterCounter++

	return networks, nil

}

type SubnetClientStub struct {
	SubnetListIterator map[string]*StubSubnetsListIterator
}

func (s *SubnetClientStub) List(ctx context.Context, req *computepb.ListSubnetworksRequest, opts ...gax.CallOption) SubnetIterator {
	project := req.Project
	return s.SubnetListIterator[project]
}

func TestCollectVpcAssets(t *testing.T) {
	for _, tt := range []struct {
		name           string
		cfg            config
		networks       map[string]*StubNetworksListIterator
		expectedEvents []beat.Event
	}{
		{
			name: "single project, multiple vpcs",
			cfg: config{
				Projects: []string{"my_project"},
			},
			networks: map[string]*StubNetworksListIterator{
				"my_project": {
					ReturnNetworksList: []*computepb.Network{
						{
							Id:   proto.Uint64(1),
							Name: proto.String("test-vpc-1"),
						},
						{
							Id:   proto.Uint64(2),
							Name: proto.String("test-vpc-2"),
						},
					},
				},
			},
			expectedEvents: []beat.Event{
				{
					Fields: mapstr.M{
						"asset.ean":        "network:1",
						"asset.id":         "1",
						"asset.name":       "test-vpc-1",
						"asset.type":       "gcp.vpc",
						"asset.kind":       "network",
						"cloud.account.id": "my_project",
						"cloud.provider":   "gcp",
					},
					Meta: mapstr.M{
						"index": "assets-gcp.vpc-default",
					},
				},
				{
					Fields: mapstr.M{
						"asset.ean":        "network:2",
						"asset.id":         "2",
						"asset.name":       "test-vpc-2",
						"asset.type":       "gcp.vpc",
						"asset.kind":       "network",
						"cloud.account.id": "my_project",
						"cloud.provider":   "gcp",
					},
					Meta: mapstr.M{
						"index": "assets-gcp.vpc-default",
					},
				},
			},
		},
		{
			name: "multiple projects, multiple vpcs",
			cfg: config{
				Projects: []string{"my_first_project", "my_second_project"},
			},
			networks: map[string]*StubNetworksListIterator{
				"my_first_project": {
					ReturnNetworksList: []*computepb.Network{
						{
							Id:   proto.Uint64(1),
							Name: proto.String("test-vpc-1"),
						},
						{
							Id:   proto.Uint64(2),
							Name: proto.String("test-vpc-2"),
						},
					},
				},
				"my_second_project": {
					ReturnNetworksList: []*computepb.Network{
						{
							Id:   proto.Uint64(3),
							Name: proto.String("test-vpc-3"),
						},
						{
							Id:   proto.Uint64(4),
							Name: proto.String("test-vpc-4"),
						},
					},
				},
			},
			expectedEvents: []beat.Event{
				{
					Fields: mapstr.M{
						"asset.ean":        "network:1",
						"asset.id":         "1",
						"asset.name":       "test-vpc-1",
						"asset.type":       "gcp.vpc",
						"asset.kind":       "network",
						"cloud.account.id": "my_first_project",
						"cloud.provider":   "gcp",
					},
					Meta: mapstr.M{
						"index": "assets-gcp.vpc-default",
					},
				},
				{
					Fields: mapstr.M{
						"asset.ean":        "network:2",
						"asset.id":         "2",
						"asset.name":       "test-vpc-2",
						"asset.type":       "gcp.vpc",
						"asset.kind":       "network",
						"cloud.account.id": "my_first_project",
						"cloud.provider":   "gcp",
					},
					Meta: mapstr.M{
						"index": "assets-gcp.vpc-default",
					},
				},
				{
					Fields: mapstr.M{
						"asset.ean":        "network:3",
						"asset.id":         "3",
						"asset.name":       "test-vpc-3",
						"asset.type":       "gcp.vpc",
						"asset.kind":       "network",
						"cloud.account.id": "my_second_project",
						"cloud.provider":   "gcp",
					},
					Meta: mapstr.M{
						"index": "assets-gcp.vpc-default",
					},
				},
				{
					Fields: mapstr.M{
						"asset.ean":        "network:4",
						"asset.id":         "4",
						"asset.name":       "test-vpc-4",
						"asset.type":       "gcp.vpc",
						"asset.kind":       "network",
						"cloud.account.id": "my_second_project",
						"cloud.provider":   "gcp",
					},
					Meta: mapstr.M{
						"index": "assets-gcp.vpc-default",
					},
				},
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			publisher := testutil.NewInMemoryPublisher()

			ctx := context.Background()
			client := NetworkClientStub{NetworkListIterator: tt.networks}
			listClient := listNetworkAPIClient{List: func(ctx context.Context, req *computepb.ListNetworksRequest, opts ...gax.CallOption) NetworkIterator {
				return client.List(ctx, req, opts...)
			}}
			err := collectVpcAssets(ctx, tt.cfg, listClient, publisher)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedEvents, publisher.Events)
		})
	}
}

func TestCollectSubnetAssets(t *testing.T) {
	for _, tt := range []struct {
		name           string
		cfg            config
		subnets        map[string]*StubSubnetsListIterator
		expectedEvents []beat.Event
	}{
		{
			name: "single project, no specified region, multiple subnets",
			cfg: config{
				Projects: []string{"my_project"},
			},
			subnets: map[string]*StubSubnetsListIterator{
				"my_project": {
					ReturnSubnetsList: []*computepb.Subnetwork{
						{
							Id:     proto.Uint64(1),
							Name:   proto.String("test-subnet-1"),
							Region: proto.String("europe-west-1"),
						},
						{
							Id:     proto.Uint64(2),
							Name:   proto.String("test-subnet-2"),
							Region: proto.String("europe-west-1"),
						},
					},
				},
			},
			expectedEvents: []beat.Event{
				{
					Fields: mapstr.M{
						"asset.ean":        "network:1",
						"asset.id":         "1",
						"asset.name":       "test-subnet-1",
						"asset.type":       "gcp.subnet",
						"asset.kind":       "network",
						"cloud.account.id": "my_project",
						"cloud.provider":   "gcp",
						"cloud.region":     "europe-west-1",
					},
					Meta: mapstr.M{
						"index": "assets-gcp.subnet-default",
					},
				},
				{
					Fields: mapstr.M{
						"asset.ean":        "network:2",
						"asset.id":         "2",
						"asset.name":       "test-subnet-2",
						"asset.type":       "gcp.subnet",
						"asset.kind":       "network",
						"cloud.account.id": "my_project",
						"cloud.provider":   "gcp",
						"cloud.region":     "europe-west-1",
					},
					Meta: mapstr.M{
						"index": "assets-gcp.subnet-default",
					},
				},
			},
		},
		{
			name: "multiple projects, specific regions, multiple subnets",
			cfg: config{
				Projects: []string{"my_first_project", "my_second_project"},
				Regions:  []string{"europe-west-1", "us-central1"},
			},
			subnets: map[string]*StubSubnetsListIterator{
				"my_first_project": {
					ReturnSubnetsList: []*computepb.Subnetwork{
						{
							Id:     proto.Uint64(1),
							Name:   proto.String("test-subnet-1"),
							Region: proto.String("europe-west-1"),
						},
						{
							Id:     proto.Uint64(2),
							Name:   proto.String("test-subnet-2"),
							Region: proto.String("europe-west-1"),
						},
					},
				},
				"my_second_project": {
					ReturnSubnetsList: []*computepb.Subnetwork{
						{
							Id:     proto.Uint64(3),
							Name:   proto.String("test-subnet-3"),
							Region: proto.String("europe-west-1"),
						},
						{
							Id:     proto.Uint64(4),
							Name:   proto.String("test-subnet-4"),
							Region: proto.String("europe-west-1"),
						},
						{
							Id:     proto.Uint64(5), //this should not appear in the events
							Name:   proto.String("test-subnet-5"),
							Region: proto.String("us-west1"),
						},
						{
							Id:     proto.Uint64(6),
							Name:   proto.String("test-subnet-6"),
							Region: proto.String("us-central1"),
						},
					},
				},
			},
			expectedEvents: []beat.Event{
				{
					Fields: mapstr.M{
						"asset.ean":        "network:1",
						"asset.id":         "1",
						"asset.name":       "test-subnet-1",
						"asset.type":       "gcp.subnet",
						"asset.kind":       "network",
						"cloud.account.id": "my_first_project",
						"cloud.provider":   "gcp",
						"cloud.region":     "europe-west-1",
					},
					Meta: mapstr.M{
						"index": "assets-gcp.subnet-default",
					},
				},
				{
					Fields: mapstr.M{
						"asset.ean":        "network:2",
						"asset.id":         "2",
						"asset.name":       "test-subnet-2",
						"asset.type":       "gcp.subnet",
						"asset.kind":       "network",
						"cloud.account.id": "my_first_project",
						"cloud.provider":   "gcp",
						"cloud.region":     "europe-west-1",
					},
					Meta: mapstr.M{
						"index": "assets-gcp.subnet-default",
					},
				},
				{
					Fields: mapstr.M{
						"asset.ean":        "network:3",
						"asset.id":         "3",
						"asset.name":       "test-subnet-3",
						"asset.type":       "gcp.subnet",
						"asset.kind":       "network",
						"cloud.account.id": "my_second_project",
						"cloud.provider":   "gcp",
						"cloud.region":     "europe-west-1",
					},
					Meta: mapstr.M{
						"index": "assets-gcp.subnet-default",
					},
				},
				{
					Fields: mapstr.M{
						"asset.ean":        "network:4",
						"asset.id":         "4",
						"asset.name":       "test-subnet-4",
						"asset.type":       "gcp.subnet",
						"asset.kind":       "network",
						"cloud.account.id": "my_second_project",
						"cloud.provider":   "gcp",
						"cloud.region":     "europe-west-1",
					},
					Meta: mapstr.M{
						"index": "assets-gcp.subnet-default",
					},
				},
				{
					Fields: mapstr.M{
						"asset.ean":        "network:6",
						"asset.id":         "6",
						"asset.name":       "test-subnet-6",
						"asset.type":       "gcp.subnet",
						"asset.kind":       "network",
						"cloud.account.id": "my_second_project",
						"cloud.provider":   "gcp",
						"cloud.region":     "us-central1",
					},
					Meta: mapstr.M{
						"index": "assets-gcp.subnet-default",
					},
				},
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			publisher := testutil.NewInMemoryPublisher()

			ctx := context.Background()
			client := SubnetClientStub{SubnetListIterator: tt.subnets}
			listClient := listSubnetworkAPIClient{List: func(ctx context.Context, req *computepb.ListSubnetworksRequest, opts ...gax.CallOption) SubnetIterator {
				return client.List(ctx, req, opts...)
			}}
			err := collectSubnetAssets(ctx, tt.cfg, listClient, publisher)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedEvents, publisher.Events)
		})
	}
}

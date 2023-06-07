package gcp

import (
	"cloud.google.com/go/compute/apiv1/computepb"
	"context"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/inputrunner/input/testutil"
	"github.com/gogo/protobuf/proto"
	"github.com/googleapis/gax-go/v2"
	"github.com/stretchr/testify/assert"
	"google.golang.org/api/iterator"
	"testing"
)

type StubNetworkListIterator struct {
	iterCounter          int
	ReturnNetworkList    []*computepb.Network
	ReturnInstancesError error
}

func (it *StubNetworkListIterator) Next() (*computepb.Network, error) {

	if it.ReturnInstancesError != nil {
		return &computepb.Network{}, it.ReturnInstancesError
	}

	if it.iterCounter == len(it.ReturnNetworkList) {
		return &computepb.Network{}, iterator.Done
	}

	networks := it.ReturnNetworkList[it.iterCounter]
	it.iterCounter++

	return networks, nil

}

type NetworkClientStub struct {
	NetworkListIterator map[string]*StubNetworkListIterator
}

func (s *NetworkClientStub) List(ctx context.Context, req *computepb.ListNetworksRequest, opts ...gax.CallOption) NetworkIterator {
	project := req.Project
	return s.NetworkListIterator[project]
}

func TestGetAllVPCs(t *testing.T) {
	for _, tt := range []struct {
		name           string
		cfg            config
		networks       map[string]*StubNetworkListIterator
		expectedEvents []beat.Event
	}{
		{
			name: "single project, multiple vpcs",
			cfg: config{
				Projects: []string{"my_project"},
			},
			networks: map[string]*StubNetworkListIterator{
				"my_project": {
					ReturnNetworkList: []*computepb.Network{
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
						"asset.ean":        "gcp.vpc:1",
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
						"asset.ean":        "gcp.vpc:2",
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
			networks: map[string]*StubNetworkListIterator{
				"my_first_project": {
					ReturnNetworkList: []*computepb.Network{
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
					ReturnNetworkList: []*computepb.Network{
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
						"asset.ean":        "gcp.vpc:1",
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
						"asset.ean":        "gcp.vpc:2",
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
						"asset.ean":        "gcp.vpc:3",
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
						"asset.ean":        "gcp.vpc:4",
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
			err := collectNetworkAssets(ctx, tt.cfg, listClient, publisher)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedEvents, publisher.Events)
		})
	}
}

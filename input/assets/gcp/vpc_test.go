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
			name: "test",
			cfg: config{
				Projects: []string{"my_project"},
			},
			networks: map[string]*StubNetworkListIterator{
				"my_project": {
					ReturnNetworkList: []*computepb.Network{
						{
							Id:   proto.Uint64(1),
							Name: proto.String("test-vpc"),
						},
					},
				},
			},
			expectedEvents: []beat.Event{
				{
					Fields: mapstr.M{
						"asset.ean":        "gcp.vpc:1",
						"asset.id":         "1",
						"asset.name":       "test-vpc",
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

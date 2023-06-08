package aws

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/stretchr/testify/assert"

	"github.com/elastic/assetbeat/input/testutil"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

var instanceID_1 = "i-1111111"
var ownerID_1 = "11111111111111"
var tag_1_k = "mykey"
var tag_1_v = "myvalue"
var subnetID1 = "mysubnetid1"
var instanceID_2 = "i-2222222"

type mockDescribeInstancesAPI func(ctx context.Context, params *ec2.DescribeInstancesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeInstancesOutput, error)

func (m mockDescribeInstancesAPI) DescribeInstances(ctx context.Context, params *ec2.DescribeInstancesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeInstancesOutput, error) {
	return m(ctx, params, optFns...)
}

func TestAssetsAWS_collectEC2Assets(t *testing.T) {
	for _, tt := range []struct {
		name           string
		region         string
		client         func(t *testing.T) ec2.DescribeInstancesAPIClient
		expectedEvents []beat.Event
	}{{
		name:   "Test with multiple EC2 instances returned",
		region: "eu-west-1",
		client: func(t *testing.T) ec2.DescribeInstancesAPIClient {
			return mockDescribeInstancesAPI(func(ctx context.Context, params *ec2.DescribeInstancesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeInstancesOutput, error) {
				t.Helper()
				return &ec2.DescribeInstancesOutput{
					NextToken: nil,
					Reservations: []types.Reservation{
						{
							OwnerId: &ownerID_1,
							Instances: []types.Instance{
								{
									InstanceId: &instanceID_1,
									State:      &types.InstanceState{Name: "running"},
									Tags: []types.Tag{
										{
											Key:   &tag_1_k,
											Value: &tag_1_v,
										},
									},
									SubnetId: &subnetID1,
								},
								{
									InstanceId: &instanceID_2,
									State:      &types.InstanceState{Name: "stopped"},
									SubnetId:   &subnetID1,
								},
							},
						},
					},
				}, nil
			})
		},
		expectedEvents: []beat.Event{
			{
				Fields: mapstr.M{
					"asset.ean":            "host:" + instanceID_1,
					"asset.id":             instanceID_1,
					"asset.metadata.state": "running",
					"asset.type":           "aws.ec2.instance",
					"asset.kind":           "host",
					"asset.parents": []string{
						"network:" + subnetID1,
					},
					"asset.metadata.tags." + tag_1_k: tag_1_v,
					"cloud.account.id":               "11111111111111",
					"cloud.provider":                 "aws",
					"cloud.region":                   "eu-west-1",
				},
				Meta: mapstr.M{
					"index": "assets-aws.ec2.instance-default",
				},
			},
			{
				Fields: mapstr.M{
					"asset.ean":            "host:" + instanceID_2,
					"asset.id":             instanceID_2,
					"asset.metadata.state": "stopped",
					"asset.type":           "aws.ec2.instance",
					"asset.kind":           "host",
					"asset.parents": []string{
						"network:" + subnetID1,
					},
					"cloud.account.id": "11111111111111",
					"cloud.provider":   "aws",
					"cloud.region":     "eu-west-1",
				},
				Meta: mapstr.M{
					"index": "assets-aws.ec2.instance-default",
				},
			},
		},
	},
	} {
		t.Run(tt.name, func(t *testing.T) {
			publisher := testutil.NewInMemoryPublisher()

			ctx := context.Background()
			logger := logp.NewLogger("test")

			err := collectEC2Assets(ctx, tt.client(t), tt.region, "", logger, publisher)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedEvents, publisher.Events)
		})

	}
}

func TestAssetsAWS_flattenEC2Tags(t *testing.T) {
	tag1, tag2, a, b := "tag1", "tag2", "a", "b"
	tags := []types.Tag{{Key: &tag1, Value: &a}, {Key: &tag2, Value: &b}}
	flat := flattenEC2Tags(tags)
	expected := mapstr.M{"tag1": "a", "tag2": "b"}
	assert.Equal(t, expected, flat)
}

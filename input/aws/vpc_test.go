package aws

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/smithy-go/middleware"
	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/inputrunner/input/testutil"
)

var vpcId1 = "vpc-id-1"
var vpcId2 = "vpc-id-2"
var isDefaultVPC = true
var isNotDefaultVPC = false

var subnetID2 = "subnet-2"

type mockDescribeVpcsAPI func(ctx context.Context, params *ec2.DescribeVpcsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeVpcsOutput, error)

func (m mockDescribeVpcsAPI) DescribeVpcs(ctx context.Context, params *ec2.DescribeVpcsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeVpcsOutput, error) {
	return m(ctx, params, optFns...)
}

func TestAssetsAWS_collectVPCAssets(t *testing.T) {
	for _, tt := range []struct {
		name           string
		region         string
		client         func(t *testing.T) ec2.DescribeVpcsAPIClient
		expectedEvents []beat.Event
	}{
		{
			name:   "Test with multiple VPCs",
			region: "eu-west-1",
			client: func(t *testing.T) ec2.DescribeVpcsAPIClient {
				return mockDescribeVpcsAPI(func(ctx context.Context, params *ec2.DescribeVpcsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeVpcsOutput, error) {
					t.Helper()
					return &ec2.DescribeVpcsOutput{
						NextToken: nil,
						Vpcs: []types.Vpc{
							{
								OwnerId: &ownerID_1,
								VpcId:   &vpcId1,
								Tags: []types.Tag{
									{
										Key:   &tag_1_k,
										Value: &tag_1_v,
									},
								},
								IsDefault: &isDefaultVPC,
							},
							{
								OwnerId:   &ownerID_1,
								VpcId:     &vpcId2,
								IsDefault: &isNotDefaultVPC,
							},
						},
						ResultMetadata: middleware.Metadata{},
					}, nil
				})
			},
			expectedEvents: []beat.Event{
				{
					Fields: mapstr.M{
						"asset.ean":                      "network:" + vpcId1,
						"asset.id":                       vpcId1,
						"asset.type":                     "aws.vpc",
						"asset.kind":                     "network",
						"asset.metadata.isDefault":       &isDefaultVPC,
						"asset.metadata.tags." + tag_1_k: tag_1_v,
						"cloud.account.id":               ownerID_1,
						"cloud.provider":                 "aws",
						"cloud.region":                   "eu-west-1",
					},
					Meta: mapstr.M{
						"index": "assets-aws.vpc-default",
					},
				},
				{
					Fields: mapstr.M{
						"asset.ean":                "network:" + vpcId2,
						"asset.id":                 vpcId2,
						"asset.type":               "aws.vpc",
						"asset.kind":               "network",
						"asset.metadata.isDefault": &isNotDefaultVPC,
						"cloud.account.id":         ownerID_1,
						"cloud.provider":           "aws",
						"cloud.region":             "eu-west-1",
					},
					Meta: mapstr.M{
						"index": "assets-aws.vpc-default",
					},
				},
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			publisher := testutil.NewInMemoryPublisher()

			ctx := context.Background()
			logger := logp.NewLogger("test")

			err := collectVPCAssets(ctx, tt.client(t), tt.region, "", logger, publisher)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedEvents, publisher.Events)
		})
	}
}

type mockDescribeSubnetsAPI func(ctx context.Context, params *ec2.DescribeSubnetsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeSubnetsOutput, error)

func (m mockDescribeSubnetsAPI) DescribeSubnets(ctx context.Context, params *ec2.DescribeSubnetsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeSubnetsOutput, error) {
	return m(ctx, params, optFns...)
}

func TestAssetsAWS_collectSubnetAssets(t *testing.T) {
	for _, tt := range []struct {
		name           string
		region         string
		client         func(t *testing.T) ec2.DescribeSubnetsAPIClient
		expectedEvents []beat.Event
	}{
		{
			name:   "Test with multiple Subnets",
			region: "eu-west-1",
			client: func(t *testing.T) ec2.DescribeSubnetsAPIClient {
				return mockDescribeSubnetsAPI(func(ctx context.Context, params *ec2.DescribeSubnetsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeSubnetsOutput, error) {
					t.Helper()
					return &ec2.DescribeSubnetsOutput{
						Subnets: []types.Subnet{
							{
								OwnerId:  &ownerID_1,
								SubnetId: &subnetID1,
								Tags: []types.Tag{
									{
										Key:   &tag_1_k,
										Value: &tag_1_v,
									},
								},
								VpcId: &vpcId1,
								State: "available",
							},
							{
								OwnerId:  &ownerID_1,
								SubnetId: &subnetID2,
								VpcId:    &vpcId1,
								State:    "pending",
							},
						},
					}, nil
				})
			},
			expectedEvents: []beat.Event{
				{
					Fields: mapstr.M{
						"asset.ean":  "network:" + subnetID1,
						"asset.id":   subnetID1,
						"asset.type": "aws.subnet",
						"asset.kind": "network",
						"asset.parents": []string{
							"network:vpc-id-1",
						},
						"asset.metadata.state":           "available",
						"asset.metadata.tags." + tag_1_k: tag_1_v,
						"cloud.account.id":               ownerID_1,
						"cloud.provider":                 "aws",
						"cloud.region":                   "eu-west-1",
					},
					Meta: mapstr.M{
						"index": "assets-aws.subnet-default",
					},
				},
				{
					Fields: mapstr.M{
						"asset.ean":  "network:" + subnetID2,
						"asset.id":   subnetID2,
						"asset.type": "aws.subnet",
						"asset.kind": "network",
						"asset.parents": []string{
							"network:vpc-id-1",
						},
						"asset.metadata.state": "pending",
						"cloud.account.id":     ownerID_1,
						"cloud.provider":       "aws",
						"cloud.region":         "eu-west-1",
					},
					Meta: mapstr.M{
						"index": "assets-aws.subnet-default",
					},
				},
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			publisher := testutil.NewInMemoryPublisher()

			ctx := context.Background()
			logger := logp.NewLogger("test")

			err := collectSubnetAssets(ctx, tt.client(t), tt.region, "", logger, publisher)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedEvents, publisher.Events)
		})
	}
}

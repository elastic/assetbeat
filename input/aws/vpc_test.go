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

package aws

import (
	"context"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/elastic/assetbeat/input/internal"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/smithy-go/middleware"
	"github.com/stretchr/testify/assert"

	"github.com/elastic/assetbeat/input/testutil"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

var vpcId1 = "vpc-id-1"
var vpcName1 = "vpc-name-1"
var vpcId2 = "vpc-id-2"
var vpcName2 = "vpc-name-2"
var isDefaultVPC = true
var isNotDefaultVPC = false

var subnetID_1 = "subnet-id-1"
var subnetName_1 = "subnet-name-1"
var subnetID_2 = "subnet-id-2"
var subnetName_2 = "subnet-name-2"

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
									{
										Key:   to.Ptr("Name"),
										Value: &vpcName1,
									},
								},
								IsDefault: &isDefaultVPC,
							},
							{
								OwnerId:   &ownerID_1,
								VpcId:     &vpcId2,
								IsDefault: &isNotDefaultVPC,
								Tags: []types.Tag{
									{
										Key:   to.Ptr("Name"),
										Value: &vpcName2,
									},
								},
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
						"asset.name":                     vpcName1,
						"asset.type":                     "aws.vpc",
						"asset.kind":                     "network",
						"asset.metadata.isDefault":       &isDefaultVPC,
						"asset.metadata.tags." + tag_1_k: tag_1_v,
						"cloud.account.id":               ownerID_1,
						"cloud.provider":                 "aws",
						"cloud.region":                   "eu-west-1",
					},
					Meta: mapstr.M{
						"index": internal.GetDefaultIndexName(),
					},
				},
				{
					Fields: mapstr.M{
						"asset.ean":                "network:" + vpcId2,
						"asset.id":                 vpcId2,
						"asset.name":               vpcName2,
						"asset.type":               "aws.vpc",
						"asset.kind":               "network",
						"asset.metadata.isDefault": &isNotDefaultVPC,
						"cloud.account.id":         ownerID_1,
						"cloud.provider":           "aws",
						"cloud.region":             "eu-west-1",
					},
					Meta: mapstr.M{
						"index": internal.GetDefaultIndexName(),
					},
				},
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			publisher := testutil.NewInMemoryPublisher()

			ctx := context.Background()
			logger := logp.NewLogger("test")

			err := collectVPCAssets(ctx, tt.client(t), tt.region, logger, publisher)
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
								SubnetId: &subnetID_1,
								Tags: []types.Tag{
									{
										Key:   &tag_1_k,
										Value: &tag_1_v,
									},
									{
										Key:   to.Ptr("Name"),
										Value: &subnetName_1,
									},
								},
								VpcId: &vpcId1,
								State: "available",
							},
							{
								OwnerId:  &ownerID_1,
								SubnetId: &subnetID_2,
								VpcId:    &vpcId1,
								Tags: []types.Tag{
									{
										Key:   to.Ptr("Name"),
										Value: &subnetName_2,
									},
								},
								State: "pending",
							},
						},
					}, nil
				})
			},
			expectedEvents: []beat.Event{
				{
					Fields: mapstr.M{
						"asset.ean":  "network:" + subnetID_1,
						"asset.id":   subnetID_1,
						"asset.name": subnetName_1,
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
						"index": internal.GetDefaultIndexName(),
					},
				},
				{
					Fields: mapstr.M{
						"asset.ean":  "network:" + subnetID_2,
						"asset.id":   subnetID_2,
						"asset.name": subnetName_2,
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
						"index": internal.GetDefaultIndexName(),
					},
				},
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			publisher := testutil.NewInMemoryPublisher()

			ctx := context.Background()
			logger := logp.NewLogger("test")

			err := collectSubnetAssets(ctx, tt.client(t), tt.region, logger, publisher)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedEvents, publisher.Events)
		})
	}
}

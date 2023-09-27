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
	"github.com/stretchr/testify/assert"

	"github.com/elastic/assetbeat/input/testutil"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

var instanceID_1 = "i-1111111"
var instanceName_1 = "test-instance-1"
var ownerID_1 = "11111111111111"
var tag_1_k = "mykey"
var tag_1_v = "myvalue"
var subnetID1 = "mysubnetid1"
var instanceID_2 = "i-2222222"
var instanceName_2 = "test-instance-2"

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
										{
											Key:   to.Ptr("Name"),
											Value: &instanceName_1,
										},
									},
									SubnetId: &subnetID1,
								},
								{
									InstanceId: &instanceID_2,
									State:      &types.InstanceState{Name: "stopped"},
									SubnetId:   &subnetID1,
									Tags: []types.Tag{
										{
											Key:   to.Ptr("Name"),
											Value: &instanceName_2,
										},
									},
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
					"asset.name":           instanceName_1,
					"asset.parents": []string{
						"network:" + subnetID1,
					},
					"asset.metadata.tags." + tag_1_k: tag_1_v,
					"cloud.account.id":               "11111111111111",
					"cloud.provider":                 "aws",
					"cloud.region":                   "eu-west-1",
				},
				Meta: mapstr.M{
					"index": internal.GetDefaultIndexName(),
				},
			},
			{
				Fields: mapstr.M{
					"asset.ean":            "host:" + instanceID_2,
					"asset.id":             instanceID_2,
					"asset.metadata.state": "stopped",
					"asset.type":           "aws.ec2.instance",
					"asset.kind":           "host",
					"asset.name":           instanceName_2,
					"asset.parents": []string{
						"network:" + subnetID1,
					},
					"cloud.account.id": "11111111111111",
					"cloud.provider":   "aws",
					"cloud.region":     "eu-west-1",
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

			err := collectEC2Assets(ctx, tt.client(t), tt.region, logger, publisher)
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

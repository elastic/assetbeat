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
	"fmt"

	"github.com/elastic/assetbeat/input/internal"
	stateless "github.com/elastic/beats/v7/filebeat/input/v2/input-stateless"

	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

func collectVPCAssets(ctx context.Context, client ec2.DescribeVpcsAPIClient, region string, log *logp.Logger, publisher stateless.Publisher) error {
	vpcs, err := describeVPCs(ctx, client)
	if err != nil {
		return err
	}

	assetType := "aws.vpc"
	assetKind := "network"
	for _, vpc := range vpcs {
		options := []internal.AssetOption{
			internal.WithAssetCloudProvider("aws"),
			internal.WithAssetRegion(region),
			internal.WithAssetAccountID(*vpc.OwnerId),
			internal.WithAssetName(getEC2AssetNameFromTags(vpc.Tags)),
			internal.WithAssetKindAndID(assetKind, *vpc.VpcId),
			internal.WithAssetType(assetType),
			WithAssetTags(flattenEC2Tags(vpc.Tags)),
			internal.WithAssetMetadata(mapstr.M{
				"isDefault": vpc.IsDefault,
			}),
		}
		vpcName := getEC2AssetNameFromTags(vpc.Tags)
		if vpcName != "" {
			options = append(options, internal.WithAssetName(vpcName))
		}
		internal.Publish(publisher, nil,
			options...,
		)
	}

	return nil
}

func collectSubnetAssets(ctx context.Context, client ec2.DescribeSubnetsAPIClient, region string, log *logp.Logger, publisher stateless.Publisher) error {
	subnets, err := describeSubnets(ctx, client)
	if err != nil {
		return err
	}

	assetType := "aws.subnet"
	assetKind := "network"
	for _, subnet := range subnets {
		var parents []string
		if subnet.VpcId != nil {
			parents = []string{"network:" + *subnet.VpcId}
		}

		options := []internal.AssetOption{
			internal.WithAssetCloudProvider("aws"),
			internal.WithAssetRegion(region),
			internal.WithAssetAccountID(*subnet.OwnerId),
			internal.WithAssetKindAndID(assetKind, *subnet.SubnetId),
			internal.WithAssetType(assetType),
			internal.WithAssetParents(parents),
			WithAssetTags(flattenEC2Tags(subnet.Tags)),
			internal.WithAssetMetadata(mapstr.M{
				"state": string(subnet.State),
			}),
		}
		subnetName := getEC2AssetNameFromTags(subnet.Tags)
		if subnetName != "" {
			options = append(options, internal.WithAssetName(subnetName))
		}
		internal.Publish(publisher, nil,
			options...,
		)
	}

	return nil
}

func describeVPCs(ctx context.Context, client ec2.DescribeVpcsAPIClient) ([]types.Vpc, error) {
	vpcs := make([]types.Vpc, 0, 100)
	paginator := ec2.NewDescribeVpcsPaginator(client, &ec2.DescribeVpcsInput{})
	for paginator.HasMorePages() {
		resp, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("error describing VPCs: %w", err)
		}

		vpcs = append(vpcs, resp.Vpcs...)
	}

	return vpcs, nil
}

func describeSubnets(ctx context.Context, client ec2.DescribeSubnetsAPIClient) ([]types.Subnet, error) {
	subnets := make([]types.Subnet, 0, 100)
	paginator := ec2.NewDescribeSubnetsPaginator(client, &ec2.DescribeSubnetsInput{})
	for paginator.HasMorePages() {
		resp, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("error describing subnets: %w", err)
		}

		subnets = append(subnets, resp.Subnets...)
	}

	return subnets, nil
}

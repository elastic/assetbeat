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
	"github.com/elastic/assetbeat/util"
	stateless "github.com/elastic/beats/v7/filebeat/input/v2/input-stateless"

	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

type EC2Instance struct {
	InstanceID   string
	InstanceName string
	OwnerID      string
	SubnetID     string
	Tags         []types.Tag
	Metadata     mapstr.M
}

func collectEC2Assets(ctx context.Context, client ec2.DescribeInstancesAPIClient, region string, log *logp.Logger, publisher stateless.Publisher) error {
	instances, err := describeEC2Instances(ctx, client)
	if err != nil {
		return err
	}

	for _, instance := range instances {
		var parents []string
		if instance.SubnetID != "" {
			parents = []string{"network:" + instance.SubnetID}
		}
		assetType := "aws.ec2.instance"
		assetKind := "host"
		options := []internal.AssetOption{
			internal.WithAssetCloudProvider("aws"),
			internal.WithAssetRegion(region),
			internal.WithAssetAccountID(instance.OwnerID),
			internal.WithAssetKindAndID(assetKind, instance.InstanceID),
			internal.WithAssetType(assetType),
			WithAssetTags(flattenEC2Tags(instance.Tags)),
			internal.WithAssetMetadata(instance.Metadata),
		}
		if parents != nil {
			options = append(options, internal.WithAssetParents(parents))
		}
		if instance.InstanceName != "" {
			options = append(options, internal.WithAssetName(instance.InstanceName))
		}
		internal.Publish(publisher, nil,
			options...,
		)
	}

	return nil
}

func describeEC2Instances(ctx context.Context, client ec2.DescribeInstancesAPIClient) ([]EC2Instance, error) {
	instances := make([]EC2Instance, 0, 100)
	paginator := ec2.NewDescribeInstancesPaginator(client, &ec2.DescribeInstancesInput{})
	for paginator.HasMorePages() {
		resp, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("error describing EC2 instances: %w", err)
		}
		for _, reservation := range resp.Reservations {
			instances = append(instances, util.Map(func(i types.Instance) EC2Instance {
				inst := EC2Instance{
					InstanceID:   *i.InstanceId,
					InstanceName: getEC2AssetNameFromTags(i.Tags),
					OwnerID:      *reservation.OwnerId,
					Tags:         i.Tags,
					Metadata: mapstr.M{
						"state": string(i.State.Name),
					},
				}
				if i.SubnetId != nil {
					inst.SubnetID = *i.SubnetId
				}
				return inst
			}, reservation.Instances)...)
		}
	}
	return instances, nil
}

// flattenEC2Tags converts the EC2 tag format to a simple `map[string]string`
func flattenEC2Tags(tags []types.Tag) mapstr.M {
	out := mapstr.M{}
	for _, t := range tags {
		//Name is already published as asset.name
		if *t.Key != "Name" {
			out[*t.Key] = *t.Value
		}
	}
	return out
}

func getEC2AssetNameFromTags(tags []types.Tag) string {
	for _, tag := range tags {
		if *tag.Key == "Name" {
			return *tag.Value
		}
	}
	return ""
}

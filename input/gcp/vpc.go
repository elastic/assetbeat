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
	"cloud.google.com/go/compute/apiv1/computepb"
	"context"
	"github.com/elastic/assetbeat/input/internal"
	stateless "github.com/elastic/beats/v7/filebeat/input/v2/input-stateless"
	"github.com/googleapis/gax-go/v2"
	"google.golang.org/api/iterator"
	"strconv"
)

type NetworkIterator interface {
	Next() (*computepb.Network, error)
}

type listNetworkAPIClient struct {
	List func(ctx context.Context, req *computepb.ListNetworksRequest, opts ...gax.CallOption) NetworkIterator
}

type SubnetIterator interface {
	Next() (*computepb.Subnetwork, error)
}

type listSubnetworkAPIClient struct {
	List func(ctx context.Context, req *computepb.ListSubnetworksRequest, opts ...gax.CallOption) SubnetIterator
}

type vpc struct {
	ID      string
	Name    string
	Account string
}

type subnet struct {
	ID      string
	Name    string
	Account string
	Region  string
}

func collectVpcAssets(ctx context.Context, cfg config, client listNetworkAPIClient, publisher stateless.Publisher) error {

	vpcs, err := getAllVPCs(ctx, cfg, client)

	if err != nil {
		return err
	}

	assetType := "gcp.vpc"
	assetKind := "network"
	indexNamespace := cfg.IndexNamespace
	for _, vpc := range vpcs {

		internal.Publish(publisher, nil,
			internal.WithAssetCloudProvider("gcp"),
			internal.WithAssetAccountID(vpc.Account),
			internal.WithAssetKindAndID(assetKind, vpc.ID),
			internal.WithAssetName(vpc.Name),
			internal.WithAssetType(assetType),
			internal.WithIndex(assetType, indexNamespace),
		)
	}
	return nil
}

func getAllVPCs(ctx context.Context, cfg config, client listNetworkAPIClient) ([]vpc, error) {
	var vpcs []vpc
	for _, project := range cfg.Projects {
		req := &computepb.ListNetworksRequest{
			Project: project,
		}

		it := client.List(ctx, req)

		for {
			v, err := it.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				return nil, err
			}
			vpcs = append(vpcs, vpc{
				ID:      strconv.FormatUint(*v.Id, 10),
				Account: project,
				Name:    *v.Name,
			})
		}
	}
	return vpcs, nil

}

func collectSubnetAssets(ctx context.Context, cfg config, client listSubnetworkAPIClient, publisher stateless.Publisher) error {

	subnets, err := getAllSubnets(ctx, cfg, client)

	if err != nil {
		return err
	}

	assetType := "gcp.subnet"
	assetKind := "network"
	indexNamespace := cfg.IndexNamespace
	for _, subnet := range subnets {

		internal.Publish(publisher, nil,
			internal.WithAssetCloudProvider("gcp"),
			internal.WithAssetAccountID(subnet.Account),
			internal.WithAssetKindAndID(assetKind, subnet.ID),
			internal.WithAssetName(subnet.Name),
			internal.WithAssetType(assetType),
			internal.WithAssetRegion(subnet.Region),
			internal.WithIndex(assetType, indexNamespace),
		)
	}
	return nil
}

func getAllSubnets(ctx context.Context, cfg config, client listSubnetworkAPIClient) ([]subnet, error) {
	var subnets []subnet
	for _, project := range cfg.Projects {
		req := &computepb.ListSubnetworksRequest{
			Project: project,
		}

		it := client.List(ctx, req)

		for {
			s, err := it.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				return nil, err
			}
			if wantSubnet(s, cfg.Regions) {
				subnets = append(subnets, subnet{
					ID:      strconv.FormatUint(*s.Id, 10),
					Account: project,
					Name:    *s.Name,
					Region:  *s.Region,
				})
			}

		}
	}
	return subnets, nil

}

func wantSubnet(s *computepb.Subnetwork, regions []string) bool {
	if len(regions) == 0 {
		return true
	}

	for _, region := range regions {
		if region == *s.Region {
			return true
		}
	}

	return false
}

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
	"strconv"

	"github.com/googleapis/gax-go/v2"

	compute "cloud.google.com/go/compute/apiv1"
	"cloud.google.com/go/compute/apiv1/computepb"
	"google.golang.org/api/iterator"

	"github.com/elastic/assetbeat/input/internal"
	stateless "github.com/elastic/beats/v7/filebeat/input/v2/input-stateless"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/go-freelru"
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
type AggregatedSubnetworkIterator interface {
	Next() (compute.SubnetworksScopedListPair, error)
}

type listSubnetworkAPIClient struct {
	AggregatedList func(ctx context.Context, req *computepb.AggregatedListSubnetworksRequest, opts ...gax.CallOption) AggregatedSubnetworkIterator
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

func collectVpcAssets(ctx context.Context, cfg config, vpcAssetCache *freelru.LRU[string, *vpc], client listNetworkAPIClient, publisher stateless.Publisher, log *logp.Logger) error {
	vpcs, err := getAllVPCs(ctx, cfg, vpcAssetCache, client)
	if err != nil {
		return err
	}
	assetType := "gcp.vpc"
	assetKind := "network"

	log.Debug("Publishing VPCs")
	for _, vpc := range vpcs {

		options := []internal.AssetOption{
			internal.WithAssetCloudProvider("gcp"),
			internal.WithAssetAccountID(vpc.Account),
			internal.WithAssetKindAndID(assetKind, vpc.ID),
			internal.WithAssetType(assetType),
		}

		if vpc.Name != "" {
			options = append(options, internal.WithAssetName(vpc.Name))
		}
		internal.Publish(publisher, nil, options...)
	}
	return nil
}

func getAllVPCs(ctx context.Context, cfg config, vpcAssetCache *freelru.LRU[string, *vpc], client listNetworkAPIClient) ([]vpc, error) {
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
			nv := vpc{
				ID:      strconv.FormatUint(*v.Id, 10),
				Account: project,
				Name:    *v.Name,
			}
			vpcs = append(vpcs, nv)
			selfLink := *v.SelfLink
			vpcAssetCache.AddWithExpire(selfLink, &nv, cfg.Period*2)
		}
	}
	return vpcs, nil

}

func collectSubnetAssets(ctx context.Context, cfg config, subnetAssetCache *freelru.LRU[string, *subnet], client listSubnetworkAPIClient, publisher stateless.Publisher, log *logp.Logger) error {
	subnets, err := getAllSubnets(ctx, cfg, subnetAssetCache, client)

	if err != nil {
		return err
	}

	assetType := "gcp.subnet"
	assetKind := "network"
	log.Debug("Publishing Subnets")
	for _, subnet := range subnets {

		options := []internal.AssetOption{
			internal.WithAssetCloudProvider("gcp"),
			internal.WithAssetAccountID(subnet.Account),
			internal.WithAssetKindAndID(assetKind, subnet.ID),
			internal.WithAssetType(assetType),
			internal.WithAssetRegion(subnet.Region),
		}

		if subnet.Name != "" {
			options = append(options, internal.WithAssetName(subnet.Name))
		}
		internal.Publish(publisher, nil, options...)
	}
	return nil
}

func getAllSubnets(ctx context.Context, cfg config, subnetAssetCache *freelru.LRU[string, *subnet], client listSubnetworkAPIClient) ([]subnet, error) {
	var subnets []subnet
	for _, project := range cfg.Projects {
		req := &computepb.AggregatedListSubnetworksRequest{
			Project: project,
		}
		it := client.AggregatedList(ctx, req)

		for {
			subnetScopedPair, err := it.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				return nil, err
			}
			region := subnetScopedPair.Key
			if wantRegion(region, cfg.Regions) {
				for _, s := range subnetScopedPair.Value.Subnetworks {
					sb := subnet{
						ID:      strconv.FormatUint(*s.Id, 10),
						Account: project,
						Name:    *s.Name,
						Region:  *s.Region,
					}
					subnets = append(subnets, sb)
					selfLink := *s.SelfLink
					subnetAssetCache.AddWithExpire(selfLink, &sb, cfg.Period*2)
				}
			}
		}
	}
	return subnets, nil

}

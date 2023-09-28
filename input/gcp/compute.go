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
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/go-freelru"
)

type AggregatedInstanceIterator interface {
	Next() (compute.InstancesScopedListPair, error)
}

type listInstanceAPIClient struct {
	AggregatedList func(ctx context.Context, req *computepb.AggregatedListInstancesRequest, opts ...gax.CallOption) AggregatedInstanceIterator
}

type computeInstance struct {
	ID       string
	Region   string
	Account  string
	VPCs     []string
	Labels   map[string]string
	Metadata mapstr.M
	RawMd    *computepb.Metadata
	Name     string
}

func collectComputeAssets(ctx context.Context, cfg config, subnetAssetCache *freelru.LRU[string, *subnet], computeAssetCache *freelru.LRU[string, *computeInstance], client listInstanceAPIClient, publisher stateless.Publisher, log *logp.Logger) error {

	instances, err := getAllComputeInstances(ctx, cfg, subnetAssetCache, computeAssetCache, client)
	if err != nil {
		return err
	}

	assetType := "gcp.compute.instance"
	assetKind := "host"
	log.Debug("Publishing GCP compute instances")

	for _, instance := range instances {
		var parents []string
		for _, vpc := range instance.VPCs {
			if len(vpc) > 0 {
				parents = append(parents, "network:"+vpc)
			}
		}
		options := []internal.AssetOption{
			internal.WithAssetCloudProvider("gcp"),
			internal.WithAssetRegion(instance.Region),
			internal.WithAssetAccountID(instance.Account),
			internal.WithAssetKindAndID(assetKind, instance.ID),
			internal.WithAssetType(assetType),
			internal.WithAssetParents(parents),
			WithAssetLabels(internal.ToMapstr(instance.Labels)),
			internal.WithAssetMetadata(instance.Metadata),
		}

		if instance.Name != "" {
			options = append(options, internal.WithAssetName(instance.Name))
		}
		internal.Publish(publisher, nil, options...)
	}

	return nil
}

func getAllComputeInstances(ctx context.Context, cfg config, subnetAssetCache *freelru.LRU[string, *subnet], computeAssetCache *freelru.LRU[string, *computeInstance], client listInstanceAPIClient) ([]computeInstance, error) {
	var instances []computeInstance

	for _, p := range cfg.Projects {
		req := &computepb.AggregatedListInstancesRequest{
			Project: p,
		}
		it := client.AggregatedList(ctx, req)

		for {
			instanceScopedPair, err := it.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				return nil, err
			}
			zone := instanceScopedPair.Key
			if wantZone(zone, cfg.Regions) {
				for _, i := range instanceScopedPair.Value.Instances {
					var subnets []string
					for _, ni := range i.NetworkInterfaces {
						subnets = append(subnets, getSubnetIdFromLink(*ni.Subnetwork, subnetAssetCache))
					}
					cI := computeInstance{
						ID:      strconv.FormatUint(*i.Id, 10),
						Region:  getRegionFromZoneURL(zone),
						Account: p,
						VPCs:    subnets,
						Labels:  i.Labels,
						Metadata: mapstr.M{
							"state": *i.Status,
						},
						RawMd: i.GetMetadata(),
						Name:  i.GetName(),
					}
					selfLink := *i.SelfLink
					computeAssetCache.AddWithExpire(selfLink, &cI, cfg.Period*2)
					instances = append(instances, cI)
				}
			}

		}
	}

	return instances, nil
}

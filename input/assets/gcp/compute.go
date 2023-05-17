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
	compute "cloud.google.com/go/compute/apiv1"
	"cloud.google.com/go/compute/apiv1/computepb"
	"context"
	"github.com/googleapis/gax-go/v2"
	"google.golang.org/api/iterator"
	"strconv"

	stateless "github.com/elastic/beats/v7/filebeat/input/v2/input-stateless"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/inputrunner/input/assets/internal"
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
}

func collectComputeAssets(ctx context.Context, cfg config, publisher stateless.Publisher) error {
	client, err := compute.NewInstancesRESTClient(ctx, buildClientOptions(cfg)...)
	if err != nil {
		return err
	}
	defer client.Close()
	listClient := listInstanceAPIClient{AggregatedList: func(ctx context.Context, req *computepb.AggregatedListInstancesRequest, opts ...gax.CallOption) AggregatedInstanceIterator {
		return client.AggregatedList(ctx, req, opts...)
	},
	}
	instances, err := getAllComputeInstances(ctx, cfg, listClient)
	if err != nil {
		return err
	}

	assetType := "gcp.compute.instance"
	indexNamespace := cfg.IndexNamespace
	for _, instance := range instances {
		var parents []string
		parents = append(parents, instance.VPCs...)

		internal.Publish(publisher,
			internal.WithAssetCloudProvider("gcp"),
			internal.WithAssetRegion(instance.Region),
			internal.WithAssetAccountID(instance.Account),
			internal.WithAssetTypeAndID(assetType, instance.ID),
			internal.WithAssetParents(parents),
			WithAssetLabels(internal.ToMapstr(instance.Labels)),
			internal.WithIndex(assetType, indexNamespace),
			internal.WithAssetMetadata(instance.Metadata),
		)
	}

	return nil
}

func getAllComputeInstances(ctx context.Context, cfg config, client listInstanceAPIClient) ([]computeInstance, error) {
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
			for _, i := range instanceScopedPair.Value.Instances {
				if wantInstance(cfg, i) {
					var vpcs []string
					for _, ni := range i.NetworkInterfaces {
						vpcs = append(vpcs, getResourceNameFromURL(*ni.Network))
					}

					instances = append(instances, computeInstance{
						ID:      strconv.FormatUint(*i.Id, 10),
						Region:  getRegionFromZoneURL(*i.Zone),
						Account: p,
						VPCs:    vpcs,
						Labels:  i.Labels,
						Metadata: mapstr.M{
							"state": *i.Status,
						},
					})
				}
			}
		}
	}

	return instances, nil
}

func wantInstance(cfg config, i *computepb.Instance) bool {
	if len(cfg.Regions) == 0 {
		return true
	}

	region := getRegionFromZoneURL(*i.Zone)
	for _, z := range cfg.Regions {
		if z == region {
			return true
		}
	}

	return false
}

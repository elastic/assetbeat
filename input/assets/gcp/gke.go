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
	container "cloud.google.com/go/container/apiv1"
	"cloud.google.com/go/container/apiv1/containerpb"
	"context"
	"fmt"
	stateless "github.com/elastic/beats/v7/filebeat/input/v2/input-stateless"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/inputrunner/input/assets/internal"
	"github.com/googleapis/gax-go/v2"
	"google.golang.org/api/iterator"
	"strconv"
	"strings"
)

type listClustersAPIClient interface {
	ListClusters(ctx context.Context, req *containerpb.ListClustersRequest, opts ...gax.CallOption) (*containerpb.ListClustersResponse, error)
}

type containerCluster struct {
	ID        string
	Region    string
	Account   string
	VPC       string
	NodePools []*containerpb.NodePool
	Labels    map[string]string
	Metadata  mapstr.M
}

func collectGKEAssets(ctx context.Context, cfg config, log *logp.Logger, publisher stateless.Publisher) error {
	client, err := container.NewClusterManagerClient(ctx)
	if err != nil {
		return err
	}
	defer client.Close()

	computeClient, err := compute.NewInstancesRESTClient(ctx, buildClientOptions(cfg)...)
	if err != nil {
		return err
	}
	defer computeClient.Close()
	listClient := listInstanceAPIClient{
		AggregatedList: func(ctx context.Context, req *computepb.AggregatedListInstancesRequest, opts ...gax.CallOption) AggregatedInstanceIterator {
			return computeClient.AggregatedList(ctx, req, opts...)
		},
	}

	clusters, err := getAllGKEClusters(ctx, cfg, client)
	if err != nil {
		return err
	}

	indexNamespace := cfg.IndexNamespace
	assetType := "k8s.cluster"
	for _, cluster := range clusters {
		var parents []string
		var children []string

		if len(cluster.VPC) > 0 {
			parents = append(parents, cluster.VPC)
		}

		instances, err := getAllInstancesForGKECluster(ctx, cluster.Account, cluster.Region, cluster.NodePools, listClient)
		if err != nil {
			log.Warnf("Error while retrieving instances for GKE cluster %s: %+v", cluster.ID, err)
		}
		if len(instances) > 0 {
			children = append(children, instances...)
		}

		internal.Publish(publisher,
			internal.WithAssetCloudProvider("gcp"),
			internal.WithAssetRegion(cluster.Region),
			internal.WithAssetAccountID(cluster.Account),
			internal.WithAssetTypeAndID(assetType, cluster.ID),
			internal.WithAssetParents(parents),
			internal.WithAssetChildren(children),
			WithAssetLabels(internal.ToMapstr(cluster.Labels)),
			internal.WithIndex(assetType, indexNamespace),
			internal.WithAssetMetadata(cluster.Metadata),
		)
	}

	return nil
}

func getGKEInstanceKubeLabels(rawMd *computepb.Metadata) map[string]string {
	mappedMd := make(map[string]string)
	for _, item := range rawMd.GetItems() {

		if item.GetKey() != "kube-labels" {
			continue
		}
		for _, entry := range strings.Split(item.GetValue(), ",") {
			parts := strings.SplitN(entry, "=", 2)
			if len(parts) != 2 {
				continue
			}
			mappedMd[parts[0]] = parts[1]
		}

	}
	return mappedMd
}

func getAllInstancesForGKECluster(ctx context.Context, project string, region string, nodePools []*containerpb.NodePool, client listInstanceAPIClient) ([]string, error) {
	var instanceIDs []string
	zoneFilter := fmt.Sprintf("zone eq .*%s.*", region)
	req := &computepb.AggregatedListInstancesRequest{
		Project: project,
		Filter:  &zoneFilter,
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
			metadata := getGKEInstanceKubeLabels(i.Metadata)
			for _, nodePool := range nodePools {
				if metadata["cloud.google.com/gke-nodepool"] == nodePool.Name {
					id := strconv.FormatUint(*i.Id, 10)
					instanceIDs = append(instanceIDs, id)
				}
			}

		}
	}

	return instanceIDs, nil
}

func makeListClusterRequests(project string, zones []string) []*containerpb.ListClustersRequest {
	var requests []*containerpb.ListClustersRequest
	if len(zones) > 0 {
		for _, zone := range zones {
			req := &containerpb.ListClustersRequest{
				Parent: fmt.Sprintf("projects/%s/locations/%s", project, zone),
			}
			requests = append(requests, req)
		}
	} else {
		req := &containerpb.ListClustersRequest{
			Parent: fmt.Sprintf("projects/%s/locations/%s", project, "-"),
		}
		requests = append(requests, req)
	}
	return requests
}

func getAllGKEClusters(ctx context.Context, cfg config, client listClustersAPIClient) ([]containerCluster, error) {
	var clusters []containerCluster
	var zones []string
	if len(cfg.Regions) > 0 {
		zones = append(zones, cfg.Regions...)
	}
	for _, p := range cfg.Projects {
		requests := makeListClusterRequests(p, zones)
		for _, req := range requests {
			list, err := client.ListClusters(ctx, req)
			if err != nil {
				return nil, err
			}

			if err != nil {
				return nil, fmt.Errorf("error retrieving clusters list for project %s: %w", p, err)
			}

			for _, c := range list.Clusters {

				clusters = append(clusters, containerCluster{
					ID:        c.Id,
					Region:    c.Location,
					Account:   p,
					VPC:       c.Network,
					NodePools: c.NodePools,
					Labels:    c.ResourceLabels,
					Metadata: mapstr.M{
						"state": c.Status.String(),
					},
				})
			}
		}
	}

	return clusters, nil
}

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
	"fmt"
	"strings"

	stateless "github.com/elastic/beats/v7/filebeat/input/v2/input-stateless"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/inputrunner/input/assets/internal"
	"google.golang.org/api/container/v1"
)

type containerCluster struct {
	ID       string
	Region   string
	Account  string
	VPC      string
	Labels   map[string]string
	Metadata mapstr.M
}

func collectGKEAssets(ctx context.Context, cfg config, publisher stateless.Publisher) error {
	svc, err := container.NewService(ctx, buildClientOptions(cfg)...)
	if err != nil {
		return err
	}

	clusters, err := getAllGKEClusters(ctx, cfg, svc)
	if err != nil {
		return err
	}

	indexNamespace := cfg.IndexNamespace
	assetType := "k8s.cluster"
	for _, cluster := range clusters {
		var parents []string
		if len(cluster.VPC) > 0 {
			parents = append(parents, cluster.VPC)
		}

		internal.Publish(publisher,
			internal.WithAssetCloudProvider("gcp"),
			internal.WithAssetRegion(cluster.Region),
			internal.WithAssetAccountID(cluster.Account),
			internal.WithAssetTypeAndID(assetType, cluster.ID),
			internal.WithAssetParents(parents),
			WithAssetLabels(internal.ToMapstr(cluster.Labels)),
			internal.WithIndex(assetType, indexNamespace),
			internal.WithAssetMetadata(cluster.Metadata),
		)
	}

	return nil
}

func getAllGKEClusters(ctx context.Context, cfg config, svc *container.Service) ([]containerCluster, error) {
	var clusters []containerCluster

	var zones = "-"
	if len(cfg.Regions) > 0 {
		zones = strings.Join(cfg.Regions, ",")
	}

	for _, p := range cfg.Projects {
		list, err := svc.Projects.Zones.Clusters.List(p, zones).Do()
		if err != nil {
			return nil, fmt.Errorf("error retrieving clusters list for project %s: %w", p, err)
		}

		for _, c := range list.Clusters {
			clusters = append(clusters, containerCluster{
				ID:      c.Id,
				Region:  getRegionFromZoneURL(c.Zone),
				Account: p,
				VPC:     c.Network,
				Labels:  c.ResourceLabels,
				Metadata: mapstr.M{
					"state": c.Status,
				},
			})
		}
	}

	return clusters, nil
}

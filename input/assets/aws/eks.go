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
	"sync"

	stateless "github.com/elastic/beats/v7/filebeat/input/v2/input-stateless"
	"github.com/elastic/inputrunner/input/assets/internal"

	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/arn"
	"github.com/aws/aws-sdk-go-v2/service/eks"
	"github.com/aws/aws-sdk-go-v2/service/eks/types"
)

func collectEKSAssets(ctx context.Context, cfg aws.Config, log *logp.Logger, publisher stateless.Publisher) error {
	client := eks.NewFromConfig(cfg)
	clusters, err := listEKSClusters(ctx, client)
	if err != nil {
		return err
	}

	for _, clusterDetail := range describeEKSClusters(log, ctx, clusters, client) {
		if clusterDetail != nil {
			var parents []string
			if clusterDetail.ResourcesVpcConfig.VpcId != nil {
				parents = []string{*clusterDetail.ResourcesVpcConfig.VpcId}
			}

			clusterARN, _ := arn.Parse(*clusterDetail.Arn)
			internal.Publish(publisher,
				internal.WithAssetCloudProvider("aws"),
				internal.WithAssetRegion(cfg.Region),
				internal.WithAssetAccountID(clusterARN.AccountID),
				internal.WithAssetTypeAndID("k8s.cluster", *clusterDetail.Arn),
				internal.WithAssetParents(parents),
				WithAssetTags(internal.ToMapstr(clusterDetail.Tags)),
				internal.WithAssetMetadata(mapstr.M{
					"status": clusterDetail.Status,
				}),
			)
		}
	}

	return nil
}

func describeEKSClusters(log *logp.Logger, ctx context.Context, clusters []string, client *eks.Client) []*types.Cluster {
	wg := &sync.WaitGroup{}
	results := make([]*types.Cluster, len(clusters))
	for i, cluster := range clusters {
		wg.Add(1)
		go func(cluster string, idx int) {
			defer wg.Done()

			resp, err := client.DescribeCluster(ctx, &eks.DescribeClusterInput{Name: &cluster})
			if err != nil {
				log.Errorf("could not describe cluster '%s': %v", cluster, err)
			}

			results[idx] = resp.Cluster
		}(cluster, i)
	}
	wg.Wait()

	return results
}

func listEKSClusters(ctx context.Context, client *eks.Client) ([]string, error) {
	clusters := make([]string, 0, 100)
	paginator := eks.NewListClustersPaginator(client, &eks.ListClustersInput{})
	for paginator.HasMorePages() {
		resp, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		clusters = append(clusters, resp.Clusters...)
	}
	return clusters, nil
}

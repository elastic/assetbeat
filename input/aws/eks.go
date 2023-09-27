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
	"sync"

	"github.com/elastic/assetbeat/input/internal"
	stateless "github.com/elastic/beats/v7/filebeat/input/v2/input-stateless"

	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/arn"
	"github.com/aws/aws-sdk-go-v2/service/autoscaling"
	"github.com/aws/aws-sdk-go-v2/service/eks"
	"github.com/aws/aws-sdk-go-v2/service/eks/types"
)

func collectEKSAssets(ctx context.Context, cfg aws.Config, log *logp.Logger, publisher stateless.Publisher) error {
	eksClient := eks.NewFromConfig(cfg)
	asgClient := autoscaling.NewFromConfig(cfg)
	clusters, err := listEKSClusters(ctx, eksClient)

	if err != nil {
		return err
	}

	for _, clusterDetail := range describeEKSClusters(log, ctx, clusters, eksClient) {
		if clusterDetail != nil {
			var parents []string
			var children []string
			if clusterDetail.ResourcesVpcConfig.VpcId != nil {
				parents = []string{"network:" + *clusterDetail.ResourcesVpcConfig.VpcId}
			}
			nodeGroups, _ := listNodeGroups(ctx, *clusterDetail.Name, eksClient)
			instances, _ := getInstanceIDsFromNodeGroup(ctx, *clusterDetail.Name, nodeGroups, eksClient, asgClient)

			for _, instance := range instances {
				children = []string{"host:" + instance}
			}

			clusterARN, _ := arn.Parse(*clusterDetail.Arn)
			assetType := "k8s.cluster"
			assetKind := "cluster"
			options := []internal.AssetOption{
				internal.WithAssetCloudProvider("aws"),
				internal.WithAssetRegion(cfg.Region),
				internal.WithAssetAccountID(clusterARN.AccountID),
				internal.WithAssetKindAndID(assetKind, *clusterDetail.Arn),
				internal.WithAssetType(assetType),
				internal.WithAssetParents(parents),
				internal.WithAssetChildren(children),
				WithAssetTags(internal.ToMapstr(clusterDetail.Tags)),
				internal.WithAssetMetadata(mapstr.M{
					"status": clusterDetail.Status,
				}),
			}
			if *clusterDetail.Name != "" {
				options = append(options, internal.WithAssetName(*clusterDetail.Name))
			}
			internal.Publish(publisher, nil,
				options...,
			)
		}
	}

	return nil
}

func listNodeGroups(ctx context.Context, clusterName string, eksClient eks.ListNodegroupsAPIClient) ([]string, error) {
	resp, err := eksClient.ListNodegroups(ctx, &eks.ListNodegroupsInput{ClusterName: &clusterName})
	if err != nil {
		return nil, fmt.Errorf("error while listing Node Groups for cluster %s: %w", clusterName, err)
	}
	return resp.Nodegroups, nil
}

// Gets the underlying EC2 Instance IDs that are assigned to an EKS Node Group.
// Note: this function returns no instance IDs if EKS Fargate is used, since they are not exposed by AWS.
func getInstanceIDsFromNodeGroup(ctx context.Context, clusterName string, nodeGroups []string, eksClient eks.DescribeNodegroupAPIClient, asgClient autoscaling.DescribeAutoScalingGroupsAPIClient) ([]string, error) {
	var result []string
	for _, nodeGroup := range nodeGroups {
		resp, err := eksClient.DescribeNodegroup(ctx, &eks.DescribeNodegroupInput{
			ClusterName:   &clusterName,
			NodegroupName: &nodeGroup,
		})
		if err != nil {
			return nil, fmt.Errorf("error while describing Node Group %s: %w", nodeGroup, err)
		}
		autoscalingGroups := resp.Nodegroup.Resources.AutoScalingGroups
		instances, _ := getInstanceIDsFromEKSAsg(ctx, autoscalingGroups, asgClient)
		result = append(result, instances...)
	}
	return result, nil
}

// Autoscaling group exists as a type both for EKS and Autoscaling, in the AWS GO SDK. Given a list of EKS Autoscaling groups, this function
// converts it to a regular Autoscaling groups list and finds the underlying EC2 instance IDs for each Autoscaling group.
func getInstanceIDsFromEKSAsg(ctx context.Context, eksAutoscalingGroups []types.AutoScalingGroup, asgClient autoscaling.DescribeAutoScalingGroupsAPIClient) ([]string, error) {
	var instances []string
	var asgs []string
	for _, eksAsg := range eksAutoscalingGroups {
		asgs = append(asgs, *eksAsg.Name)
	}
	asgDetails, err := asgClient.DescribeAutoScalingGroups(ctx, &autoscaling.DescribeAutoScalingGroupsInput{AutoScalingGroupNames: asgs})
	if err != nil {
		return nil, fmt.Errorf("error while describing Autoscaling groups %q: %w", asgs, err)
	}
	for _, asgDetail := range asgDetails.AutoScalingGroups {
		for _, instance := range asgDetail.Instances {
			instances = append(instances, *instance.InstanceId)
		}
	}
	return instances, nil
}

func describeEKSClusters(log *logp.Logger, ctx context.Context, clusters []string, client eks.DescribeClusterAPIClient) []*types.Cluster {
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

func listEKSClusters(ctx context.Context, client eks.ListClustersAPIClient) ([]string, error) {
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

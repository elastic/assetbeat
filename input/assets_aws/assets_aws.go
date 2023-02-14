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

package assets_aws

import (
	"context"
	"fmt"
	"sync"
	"time"

	input "github.com/elastic/inputrunner/input/v2"
	stateless "github.com/elastic/inputrunner/input/v2/input-stateless"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/feature"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/go-concert/ctxtool"

	"github.com/aws/aws-sdk-go-v2/aws"
	aws_config "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	types_ec2 "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/eks"
	types_eks "github.com/aws/aws-sdk-go-v2/service/eks/types"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

func Plugin() input.Plugin {
	return input.Plugin{
		Name:       "assets_aws",
		Stability:  feature.Stable,
		Deprecated: false,
		Info:       "assets_aws",
		Manager:    stateless.NewInputManager(configure),
	}
}

func configure(cfg *conf.C) (stateless.Input, error) {
	config := defaultConfig()
	if err := cfg.Unpack(&config); err != nil {
		return nil, err
	}

	return newAssetsAWS(config)
}

func newAssetsAWS(config config) (*assetsAWS, error) {
	return &assetsAWS{config: config}, nil
}

type Config struct {
	Regions         []string      `config:"regions"`
	AccessKeyId     string        `config:"access_key_id"`
	SecretAccessKey string        `config:"secret_access_key"`
	SessionToken    string        `config:"session_token"`
	Period          time.Duration `config:"period"`
}

func defaultConfig() config {
	return config{
		Config: Config{
			Regions:         []string{"eu-west-2"},
			AccessKeyId:     "",
			SecretAccessKey: "",
			SessionToken:    "",
			Period:          time.Second * 600,
		},
	}
}

type assetsAWS struct {
	config
	accountID string
}

type config struct {
	Config `config:",inline"`
}

func (s *assetsAWS) Name() string { return "assets_aws" }

func (s *assetsAWS) Test(_ input.TestContext) error {
	return nil
}

func (s *assetsAWS) Run(inputCtx input.Context, publisher stateless.Publisher) error {
	ctx := ctxtool.FromCanceller(inputCtx.Cancelation)
	log := inputCtx.Logger.With("assets_aws")

	log.Info("aws asset collector run started")
	defer log.Info("aws asset collector run stopped")

	regions := s.Config.Regions
	credentialsProvider := credentials.StaticCredentialsProvider{
		Value: aws.Credentials{
			AccessKeyID: s.Config.AccessKeyId, SecretAccessKey: s.Config.SecretAccessKey, SessionToken: s.Config.SessionToken,
			Source: "inputrunner configuration",
		},
	}

	cfg, err := aws_config.LoadDefaultConfig(
		ctx,
		aws_config.WithCredentialsProvider(credentialsProvider),
	)
	if err != nil {
		return fmt.Errorf("failed to create AWS config: %w", err)
	}

	s.accountID, err = getAccountID(ctx, cfg)
	if err != nil {
        // not a fatal error - just leave the account ID blank
		log.Errorf("could not get account ID - it will not be set in reported assets")
	}

	// run an initial collection right away
	select {
	case <-ctx.Done():
		return nil
	default:
		s.collectAWSAssets(ctx, cfg, regions, log, publisher)
	}

	// followed by subsequent collections based on configured period
	ticker := time.NewTicker(s.config.Config.Period)
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			s.collectAWSAssets(ctx, cfg, regions, log, publisher)
		}
	}
}

func getAccountID(ctx context.Context, cfg aws.Config) (string, error) {
	stsClient := sts.NewFromConfig(cfg)
	callerID, err := stsClient.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil || callerID.Account == nil {
		return "", err
	}
	return *callerID.Account, nil
}

func (s *assetsAWS) collectAWSAssets(ctx context.Context, cfgWithCredentials aws.Config, regions []string, log *logp.Logger, publisher stateless.Publisher) {
	for _, region := range regions {
		cfg := cfgWithCredentials.Copy()
		cfg.Region = region

		go s.collectEKSAssets(ctx, cfg, log, publisher)
		go s.collectEC2Assets(ctx, cfg, log, publisher)
		go s.collectVPCAssets(ctx, cfg, log, publisher)
		go s.collectSubnetAssets(ctx, cfg, log, publisher)
	}
}

func (s *assetsAWS) collectEC2Assets(ctx context.Context, cfg aws.Config, log *logp.Logger, publisher stateless.Publisher) {
	client := ec2.NewFromConfig(cfg)
	instances, err := describeEC2Instances(ctx, client)
	if err != nil {
		log.Errorf("could not describe EC2 instances for %s: %v", cfg.Region, err)
		return
	}

	for _, instance := range instances {
		var parents []string
		if instance.SubnetId != nil {
			parents = []string{*instance.SubnetId}
		}
		s.publishAWSAsset(publisher, cfg.Region, "aws.ec2.instance", *instance.InstanceId, parents, nil, mapstr.M{
			"tags":  instance.Tags,
			"state": string(instance.State.Name),
		})
	}
}

func (s *assetsAWS) collectVPCAssets(ctx context.Context, cfg aws.Config, log *logp.Logger, publisher stateless.Publisher) {
	client := ec2.NewFromConfig(cfg)
	vpcs, err := describeVPCs(ctx, client)
	if err != nil {
		log.Errorf("could not describe VPCs for %s: %v", cfg.Region, err)
		return
	}

	for _, vpc := range vpcs {
		s.publishAWSAsset(publisher, cfg.Region, "aws.vpc", *vpc.VpcId, nil, nil, mapstr.M{
			"tags":      vpc.Tags,
			"isDefault": vpc.IsDefault,
		})
	}
}

func (s *assetsAWS) collectSubnetAssets(ctx context.Context, cfg aws.Config, log *logp.Logger, publisher stateless.Publisher) {
	client := ec2.NewFromConfig(cfg)
	subnets, err := describeSubnets(ctx, client)
	if err != nil {
		log.Errorf("could not describe Subnets for %s: %v", cfg.Region, err)
		return
	}

	for _, subnet := range subnets {
		s.publishAWSAsset(publisher, cfg.Region, "aws.subnet", *subnet.SubnetId, []string{*subnet.VpcId}, nil, mapstr.M{
			"tags":  subnet.Tags,
			"state": string(subnet.State),
		})
	}
}

func (s *assetsAWS) collectEKSAssets(ctx context.Context, cfg aws.Config, log *logp.Logger, publisher stateless.Publisher) {
	client := eks.NewFromConfig(cfg)
	clusters, err := listEKSClusters(ctx, client)
	if err != nil {
		log.Errorf("could not list EKS clusters for %s: %v", cfg.Region, err)
		return
	}

	for _, clusterDetail := range describeEKSClusters(log, ctx, clusters, client) {
		if clusterDetail != nil {
			var parents []string
			if clusterDetail.ResourcesVpcConfig.VpcId != nil {
				parents = []string{*clusterDetail.ResourcesVpcConfig.VpcId}
			}
			s.publishAWSAsset(publisher, cfg.Region, "k8s.cluster", *clusterDetail.Arn, parents, nil, mapstr.M{
				"tags":   clusterDetail.Tags,
				"status": clusterDetail.Status,
			})
		}
	}
}

func describeEKSClusters(log *logp.Logger, ctx context.Context, clusters []string, client *eks.Client) []*types_eks.Cluster {
	wg := &sync.WaitGroup{}
	results := make([]*types_eks.Cluster, len(clusters))
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

func describeVPCs(ctx context.Context, client *ec2.Client) ([]types_ec2.Vpc, error) {
	vpcs := make([]types_ec2.Vpc, 0, 100)
	paginator := ec2.NewDescribeVpcsPaginator(client, &ec2.DescribeVpcsInput{})
	for paginator.HasMorePages() {
		resp, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		vpcs = append(vpcs, resp.Vpcs...)
	}

	return vpcs, nil
}

func describeSubnets(ctx context.Context, client *ec2.Client) ([]types_ec2.Subnet, error) {
	subnets := make([]types_ec2.Subnet, 0, 100)
	paginator := ec2.NewDescribeSubnetsPaginator(client, &ec2.DescribeSubnetsInput{})
	for paginator.HasMorePages() {
		resp, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		subnets = append(subnets, resp.Subnets...)
	}

	return subnets, nil
}

func describeEC2Instances(ctx context.Context, client *ec2.Client) ([]types_ec2.Instance, error) {
	instances := make([]types_ec2.Instance, 0, 100)
	paginator := ec2.NewDescribeInstancesPaginator(client, &ec2.DescribeInstancesInput{})
	for paginator.HasMorePages() {
		resp, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, reservation := range resp.Reservations {
			instances = append(instances, reservation.Instances...)
		}
	}
	return instances, nil
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

func (s *assetsAWS) publishAWSAsset(publisher stateless.Publisher, region, assetType, assetId string, parents, children []string, metadata mapstr.M) {
	asset := mapstr.M{
		"cloud.provider": "aws",
		"cloud.region":   region,
        "cloud.account.id": s.accountID,

		"asset.type": assetType,
		"asset.id":   assetId,
		"asset.ean":  fmt.Sprintf("%s:%s", assetType, assetId),
	}

	if parents != nil {
		asset["asset.parents"] = parents
	}

	if children != nil {
		asset["asset.children"] = children
	}

	if metadata != nil {
		asset["asset.metadata"] = metadata
	}

	publisher.Publish(beat.Event{Fields: asset})
}

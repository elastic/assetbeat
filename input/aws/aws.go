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
	"time"

	"github.com/aws/aws-sdk-go-v2/service/ec2"

	stateless "github.com/elastic/beats/v7/filebeat/input/v2/input-stateless"

	"github.com/elastic/assetbeat/input/internal"
	input "github.com/elastic/beats/v7/filebeat/input/v2"

	"github.com/elastic/beats/v7/libbeat/feature"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/go-concert/ctxtool"

	"github.com/aws/aws-sdk-go-v2/aws"
	aws_config "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
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

func configure(inputCfg *conf.C) (stateless.Input, error) {
	cfg := defaultConfig()
	if err := inputCfg.Unpack(&cfg); err != nil {
		return nil, err
	}

	return newAssetsAWS(cfg)
}

func newAssetsAWS(cfg config) (*assetsAWS, error) {
	return &assetsAWS{cfg}, nil
}

type config struct {
	internal.BaseConfig `config:",inline"`
	Regions             []string `config:"regions"`
	AccessKeyId         string   `config:"access_key_id"`
	SecretAccessKey     string   `config:"secret_access_key"`
	SessionToken        string   `config:"session_token"`
}

func defaultConfig() config {
	return config{
		BaseConfig: internal.BaseConfig{
			Period:     time.Second * 600,
			AssetTypes: nil,
		},
		Regions:         []string{"eu-west-2"},
		AccessKeyId:     "",
		SecretAccessKey: "",
		SessionToken:    "",
	}
}

type assetsAWS struct {
	Config config
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

	cfg := s.Config
	period := cfg.Period

	ticker := time.NewTicker(period)
	select {
	case <-ctx.Done():
		return nil
	default:
		collectAWSAssets(ctx, log, cfg, publisher)
	}
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			collectAWSAssets(ctx, log, cfg, publisher)
		}
	}
}

func getAWSConfigForRegion(ctx context.Context, cfg config, region string) (aws.Config, error) {
	var options []func(*aws_config.LoadOptions) error
	if cfg.AccessKeyId != "" && cfg.SecretAccessKey != "" {
		credentialsProvider := credentials.StaticCredentialsProvider{
			Value: aws.Credentials{
				AccessKeyID: cfg.AccessKeyId, SecretAccessKey: cfg.SecretAccessKey, SessionToken: cfg.SessionToken,
				Source: "assetbeat configuration",
			},
		}
		options = append(options, aws_config.WithCredentialsProvider(credentialsProvider))
	}
	options = append(options, aws_config.WithRegion(region))

	return aws_config.LoadDefaultConfig(ctx, options...)
}

func collectAWSAssets(ctx context.Context, log *logp.Logger, cfg config, publisher stateless.Publisher) {
	for _, region := range cfg.Regions {
		awsCfg, err := getAWSConfigForRegion(ctx, cfg, region)
		if err != nil {
			log.Errorf("failed to create AWS config for %s: %v", region, err)
			continue
		}

		// these strings need careful documentation
		if internal.IsTypeEnabled(cfg.AssetTypes, "k8s.cluster") {
			go func() {
				err := collectEKSAssets(ctx, awsCfg, log, publisher)
				if err != nil {
					log.Errorf("error collecting EKS assets: %w", err)
				}
			}()
		}
		if internal.IsTypeEnabled(cfg.AssetTypes, "aws.ec2.instance") {
			ec2Region := region
			go func() {
				client := ec2.NewFromConfig(awsCfg)
				err := collectEC2Assets(ctx, client, ec2Region, log, publisher)
				if err != nil {
					log.Errorf("error collecting EC2 assets: %w", err)
				}
			}()
		}
		if internal.IsTypeEnabled(cfg.AssetTypes, "aws.vpc") {
			vpcRegion := region
			go func() {
				client := ec2.NewFromConfig(awsCfg)
				err := collectVPCAssets(ctx, client, vpcRegion, log, publisher)
				if err != nil {
					log.Errorf("error collecting VPC assets: %w", err)
				}
			}()
		}
		if internal.IsTypeEnabled(cfg.AssetTypes, "aws.subnet") {
			subnetRegion := region
			go func() {
				client := ec2.NewFromConfig(awsCfg)
				err := collectSubnetAssets(ctx, client, subnetRegion, log, publisher)
				if err != nil {
					log.Errorf("error collecting Subnet assets: %w", err)
				}
			}()
		}
	}
}

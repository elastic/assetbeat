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

	input "github.com/elastic/inputrunner/input/v2"
	stateless "github.com/elastic/inputrunner/input/v2/input-stateless"

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

func configure(cfg *conf.C) (stateless.Input, error) {
	config := defaultConfig()
	if err := cfg.Unpack(&config); err != nil {
		return nil, err
	}

	return newAssetsAWS(config)
}

func newAssetsAWS(config config) (*assetsAWS, error) {
	return &assetsAWS{config}, nil
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

	config := s.Config
	regions := config.Regions
	period := config.Period

	ticker := time.NewTicker(period)
	select {
	case <-ctx.Done():
		return nil
	default:
		collectAWSAssets(ctx, regions, log, config, publisher)
	}
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			collectAWSAssets(ctx, regions, log, config, publisher)
		}
	}
}

func getAWSConfigForRegion(ctx context.Context, config Config, region string) (aws.Config, error) {
	var options []func(*aws_config.LoadOptions) error
	if config.AccessKeyId != "" && config.SecretAccessKey != "" {
		credentialsProvider := credentials.StaticCredentialsProvider{
			Value: aws.Credentials{
				AccessKeyID: config.AccessKeyId, SecretAccessKey: config.SecretAccessKey, SessionToken: config.SessionToken,
				Source: "inputrunner configuration",
			},
		}
		options = append(options, aws_config.WithCredentialsProvider(credentialsProvider))
	}
	options = append(options, aws_config.WithRegion(region))

	cfg, err := aws_config.LoadDefaultConfig(
		ctx,
		options...,
	)
	return cfg, err
}

func collectAWSAssets(ctx context.Context, regions []string, log *logp.Logger, config Config, publisher stateless.Publisher) {
	for _, region := range regions {
		cfg, err := getAWSConfigForRegion(ctx, config, region)
		if err != nil {
			log.Errorf("failed to create AWS config for %s: %v", region, err)
			continue
		}

		go collectEKSAssets(ctx, cfg, log, publisher)
		go collectEC2Assets(ctx, cfg, log, publisher)
		go collectVPCAssets(ctx, cfg, log, publisher)
		go collectSubnetAssets(ctx, cfg, log, publisher)
	}
}

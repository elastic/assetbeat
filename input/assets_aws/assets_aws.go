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

	regions := s.Config.Regions
	credentialsProvider := credentials.StaticCredentialsProvider{
		Value: aws.Credentials{
			AccessKeyID: s.Config.AccessKeyId, SecretAccessKey: s.Config.SecretAccessKey, SessionToken: s.Config.SessionToken,
			Source: "inputrunner configuration",
		},
	}

	ticker := time.NewTicker(s.config.Config.Period)
	select {
	case <-ctx.Done():
		return nil
	default:
		collectAWSAssets(ctx, regions, log, credentialsProvider, publisher)
	}
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			collectAWSAssets(ctx, regions, log, credentialsProvider, publisher)
		}
	}
}

func collectAWSAssets(ctx context.Context, regions []string, log *logp.Logger, credentialsProvider aws.CredentialsProvider, publisher stateless.Publisher) {
	for _, region := range regions {
		cfg, err := aws_config.LoadDefaultConfig(
			ctx,
			aws_config.WithRegion(region),
			aws_config.WithCredentialsProvider(credentialsProvider),
		)
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

func publishAWSAsset(publisher stateless.Publisher, region, account, assetType, assetId string, parents, children []string, tags map[string]string, metadata mapstr.M) {
	asset := mapstr.M{
		"cloud.provider":   "aws",
		"cloud.region":     region,
		"cloud.account.id": account,

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

	assetMetadata := mapstr.M{}
	if tags != nil {
		assetMetadata["tags"] = tags
	}
	assetMetadata.Update(metadata)
	if len(assetMetadata) != 0 {
		asset["asset.metadata"] = assetMetadata
	}

	publisher.Publish(beat.Event{Fields: asset})
}

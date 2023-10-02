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

package azure

import (
	"context"
	"fmt"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v5"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/subscription/armsubscription"
	"github.com/elastic/assetbeat/input/internal"
	input "github.com/elastic/beats/v7/filebeat/input/v2"
	stateless "github.com/elastic/beats/v7/filebeat/input/v2/input-stateless"
	"github.com/elastic/beats/v7/libbeat/feature"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/go-concert/ctxtool"
)

func Plugin() input.Plugin {
	return input.Plugin{
		Name:       "assets_azure",
		Stability:  feature.Stable,
		Deprecated: false,
		Info:       "assets_azure",
		Manager:    stateless.NewInputManager(configure),
	}
}

func configure(inputCfg *conf.C) (stateless.Input, error) {
	cfg := defaultConfig()
	if err := inputCfg.Unpack(&cfg); err != nil {
		return nil, err
	}

	return newAssetsAzure(cfg)
}

func newAssetsAzure(cfg config) (*assetsAzure, error) {
	return &assetsAzure{cfg}, nil
}

type config struct {
	internal.BaseConfig `config:",inline"`
	Regions             []string `config:"regions"`
	ClientID            string   `config:"client_id"`
	ClientSecret        string   `config:"client_secret"`
	SubscriptionID      string   `config:"subscription_id"`
	TenantID            string   `config:"tenant_id"`
}

func defaultConfig() config {
	return config{
		BaseConfig: internal.BaseConfig{
			Period:     time.Second * 600,
			AssetTypes: nil,
		},
		Regions:        []string{},
		ClientID:       "",
		ClientSecret:   "",
		SubscriptionID: "",
		TenantID:       "",
	}
}

type assetsAzure struct {
	Config config
}

func (s *assetsAzure) Name() string { return "assets_azure" }

func (s *assetsAzure) Test(_ input.TestContext) error {
	return nil
}

func (s *assetsAzure) Run(inputCtx input.Context, publisher stateless.Publisher) error {
	ctx := ctxtool.FromCanceller(inputCtx.Cancelation)
	log := inputCtx.Logger.With("assets_azure")

	log.Info("azure asset collector run started")
	defer log.Info("azure asset collector run stopped")

	cfg := s.Config
	period := cfg.Period

	ticker := time.NewTicker(period)
	select {
	case <-ctx.Done():
		return nil
	default:
		collectAzureAssets(ctx, log, cfg, publisher)
	}
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			collectAzureAssets(ctx, log, cfg, publisher)
		}
	}
}

func getAzureCredentials(cfg config, log *logp.Logger) (azcore.TokenCredential, error) {
	if cfg.TenantID != "" && cfg.ClientID != "" && cfg.ClientSecret != "" {
		log.Debug("Retrieving Azure credentials from assetbeat configuration...")
		return azidentity.NewClientSecretCredential(cfg.TenantID, cfg.ClientID, cfg.ClientSecret, nil)
	} else {
		log.Debug("No Client or Tenant configuration provided. Retrieving default Azure credentials")
		return azidentity.NewDefaultAzureCredential(nil)
	}
}

func collectAzureAssets(ctx context.Context, log *logp.Logger, cfg config, publisher stateless.Publisher) {
	cred, err := getAzureCredentials(cfg, log)
	if err != nil {
		log.Errorf("Error while retrieving Azure credentials: %v")
	}
	subscriptions, err := getAzureSubscriptions(ctx, cfg, cred)
	if err != nil {
		log.Errorf("Error while retrieving Azure subscriptions list: %v")
	}

	for _, sub := range subscriptions {
		if internal.IsTypeEnabled(cfg.AssetTypes, "azure.vm.instance") {
			clientFactory, err := armcompute.NewClientFactory(sub, cred, nil)
			if err != nil {
				log.Errorf("Error creating Azure Compute Client Factory: %v", err)
				return
			}
			client := clientFactory.NewVirtualMachinesClient()
			go func(currentSub string) {
				err = collectAzureVMAssets(ctx, client, currentSub, cfg.Regions, log, publisher)
				if err != nil {
					log.Errorf("Error while collecting Azure VM assets: %v", err)
				}
			}(sub)
			vmClient := clientFactory.NewVirtualMachineScaleSetVMsClient()
			scaleSetsClient := clientFactory.NewVirtualMachineScaleSetsClient()
			go func(currentSub string) {
				err = collectAzureScaleSetsVMAssets(ctx, vmClient, scaleSetsClient, currentSub, cfg.Regions, log, publisher)
				if err != nil {
					log.Errorf("Error while collecting Azure Scale Sets VM assets: %v", err)
				}
			}(sub)
		}
	}
}

func getAzureSubscriptions(ctx context.Context, cfg config, cred azcore.TokenCredential) ([]string, error) {
	var subscriptions []string
	if cfg.SubscriptionID != "" {
		subscriptions = append(subscriptions, cfg.SubscriptionID)
	} else {
		subscriptionClientFactory, _ := armsubscription.NewClientFactory(cred, nil)
		client := subscriptionClientFactory.NewSubscriptionsClient()

		pager := client.NewListPager(nil)

		for pager.More() {
			page, err := pager.NextPage(ctx)
			if err != nil {
				return nil, fmt.Errorf("failed to advance page: %v", err)
			}
			for _, v := range page.Value {
				subscriptions = append(subscriptions, *v.SubscriptionID)
			}
		}
	}

	return subscriptions, nil
}

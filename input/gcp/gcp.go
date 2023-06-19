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
	"time"

	compute "cloud.google.com/go/compute/apiv1"
	"cloud.google.com/go/compute/apiv1/computepb"
	container "cloud.google.com/go/container/apiv1"
	"github.com/googleapis/gax-go/v2"
	"google.golang.org/api/option"

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
		Name:       "assets_gcp",
		Stability:  feature.Experimental,
		Deprecated: false,
		Info:       "assets_gcp",
		Manager:    stateless.NewInputManager(configure),
	}
}

func configure(cfg *conf.C) (stateless.Input, error) {
	config := defaultConfig()
	if err := cfg.Unpack(&config); err != nil {
		return nil, err
	}

	return newAssetsGCP(config)
}

func newAssetsGCP(config config) (*assetsGCP, error) {
	return &assetsGCP{config}, nil
}

type config struct {
	internal.BaseConfig `config:",inline"`
	Projects            []string `config:"projects"`
	Regions             []string `config:"regions"`
	CredsFilePath       string   `config:"credentials_file_path"`
}

func defaultConfig() config {
	return config{
		BaseConfig: internal.BaseConfig{
			Period: time.Second * 600,
		},
	}
}

type assetsGCP struct {
	config
}

func (s *assetsGCP) Name() string { return "assets_gcp" }

func (s *assetsGCP) Test(_ input.TestContext) error {
	return nil
}

func (s *assetsGCP) Run(inputCtx input.Context, publisher stateless.Publisher) error {
	ctx := ctxtool.FromCanceller(inputCtx.Cancelation)
	log := inputCtx.Logger.With("assets_gcp")

	log.Info("gcp asset collector run started")
	defer log.Info("gcp asset collector run stopped")

	ticker := time.NewTicker(s.Period)
	select {
	case <-ctx.Done():
		return nil
	default:
		err := s.collectAll(ctx, log, publisher)
		if err != nil {
			log.Errorf("error collecting assets: %w", err)
		}
	}
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			err := s.collectAll(ctx, log, publisher)
			if err != nil {
				log.Errorf("error collecting assets: %w", err)
			}
		}
	}
}

func (s *assetsGCP) collectAll(ctx context.Context, log *logp.Logger, publisher stateless.Publisher) error {
	if internal.IsTypeEnabled(s.config.AssetTypes, "gcp.compute.instance") {
		go func() {
			client, err := compute.NewInstancesRESTClient(ctx, buildClientOptions(s.config)...)
			if err != nil {
				log.Errorf("error collecting compute assets: %+v", err)
			}
			defer func() {
				if client != nil {
					client.Close()
				}
			}()
			listClient := listInstanceAPIClient{
				AggregatedList: func(ctx context.Context, req *computepb.AggregatedListInstancesRequest, opts ...gax.CallOption) AggregatedInstanceIterator {
					return client.AggregatedList(ctx, req, opts...)
				},
			}
			err = collectComputeAssets(ctx, s.config, listClient, publisher)
			if err != nil {
				log.Errorf("error collecting compute assets: %+v", err)
			}
		}()
	}
	if internal.IsTypeEnabled(s.config.AssetTypes, "k8s.cluster") {
		go func() {
			client, err := container.NewClusterManagerClient(ctx)
			if err != nil {
				log.Errorf("error collecting GKE assets: %+v", err)
			}

			computeClient, err := compute.NewInstancesRESTClient(ctx, buildClientOptions(s.config)...)
			if err != nil {
				log.Errorf("error collecting GKE assets: %+v", err)
			}
			defer func() {
				if client != nil {
					client.Close()
				}
				if computeClient != nil {
					computeClient.Close()
				}
			}()
			listClient := listInstanceAPIClient{
				AggregatedList: func(ctx context.Context, req *computepb.AggregatedListInstancesRequest, opts ...gax.CallOption) AggregatedInstanceIterator {
					return computeClient.AggregatedList(ctx, req, opts...)
				},
			}
			err = collectGKEAssets(ctx, s.config, log, listClient, client, publisher)
			if err != nil {
				log.Errorf("error collecting GKE assets: %+v", err)
			}
		}()
	}
	if internal.IsTypeEnabled(s.config.AssetTypes, "gcp.vpc") {
		go func() {
			client, err := compute.NewNetworksRESTClient(ctx, buildClientOptions(s.config)...)
			if err != nil {
				log.Errorf("error collecting GKE assets: %+v", err)
			}
			defer func() {
				if client != nil {
					client.Close()
				}
			}()
			listClient := listNetworkAPIClient{List: func(ctx context.Context, req *computepb.ListNetworksRequest, opts ...gax.CallOption) NetworkIterator {
				return client.List(ctx, req, opts...)
			}}
			err = collectVpcAssets(ctx, s.config, listClient, publisher)
			if err != nil {
				log.Errorf("error collecting GKE assets: %+v", err)
			}
		}()
	}
	if internal.IsTypeEnabled(s.config.AssetTypes, "gcp.subnet") {
		go func() {
			client, err := compute.NewSubnetworksRESTClient(ctx, buildClientOptions(s.config)...)
			if err != nil {
				log.Errorf("error collecting GKE assets: %+v", err)
			}
			defer func() {
				if client != nil {
					client.Close()
				}
			}()
			listClient := listSubnetworkAPIClient{List: func(ctx context.Context, req *computepb.ListSubnetworksRequest, opts ...gax.CallOption) SubnetIterator {
				return client.List(ctx, req, opts...)
			}}
			err = collectSubnetAssets(ctx, s.config, listClient, publisher)
			if err != nil {
				log.Errorf("error collecting GKE assets: %+v", err)
			}
		}()
	}
	return nil
}

func buildClientOptions(cfg config) []option.ClientOption {
	var opts []option.ClientOption

	if cfg.CredsFilePath != "" {
		opts = append(opts, option.WithCredentialsFile(cfg.CredsFilePath))
	}

	return opts
}

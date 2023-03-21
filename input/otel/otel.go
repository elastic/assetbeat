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

package otel

import (
	"context"
	"fmt"
	"os"

	"github.com/elastic/beats/v7/libbeat/feature"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/go-concert/ctxtool"
	input "github.com/elastic/inputrunner/input/v2"
	stateless "github.com/elastic/inputrunner/input/v2/input-stateless"

	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/dockerstatsreceiver"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/receiver/receivertest"
)

const (
	inputName = "otel"
)

func Plugin() input.Plugin {
	return input.Plugin{
		Name:       inputName,
		Stability:  feature.Experimental,
		Deprecated: false,
		Info:       "otel",
		Manager:    stateless.NewInputManager(configure),
	}
}

func configure(inputCfg *conf.C) (stateless.Input, error) {
	cfg := defaultConfig()
	if err := inputCfg.Unpack(&cfg); err != nil {
		return nil, err
	}

	return newInput(cfg)
}

func newInput(cfg config) (stateless.Input, error) {
	var f receiver.Factory

	switch cfg.ReceiverName {
	case "docker_stats":
		f = dockerstatsreceiver.NewFactory()
	default:
		return nil, fmt.Errorf("unknown receiver name %q", cfg.ReceiverName)
	}

	return &otelInput{cfg, f}, nil
}

type config struct {
	ReceiverName   string           `config:"receiver_name"`
	ReceiverConfig component.Config `config:"receiver_config"`
}

func defaultConfig() config {
	return config{}
}

type otelInput struct {
	Config config

	factory receiver.Factory
}

func (s *otelInput) Name() string { return inputName }

func (s *otelInput) Test(_ input.TestContext) error {
	return nil
}

func (s *otelInput) Run(inputCtx input.Context, publisher stateless.Publisher) error {
	ctx := ctxtool.FromCanceller(inputCtx.Cancelation)
	set := receivertest.NewNopCreateSettings()

	cfg := s.Config.ReceiverConfig
	if cfg == nil {
		cfg = s.factory.CreateDefaultConfig()
	}

	consumer, err := consumer.NewMetrics(func(ctx context.Context, ld pmetric.Metrics) error {
		// TODO: parse the metrics and do something with them so we can publish
		fmt.Fprintf(os.Stdout, "%d - %d\n", ld.MetricCount(), ld.DataPointCount())
		return nil
	})
	if err != nil {
		return err
	}

	rcv, err := s.factory.CreateMetricsReceiver(ctx, set, cfg, consumer)
	if err != nil {
		return err
	}

	err = rcv.Start(ctx, s)
	if err != nil {
		return err
	}

	<-ctx.Done()
	return rcv.Shutdown(ctx)
}

func (s *otelInput) ReportFatalError(err error) {}
func (s *otelInput) GetFactory(kind component.Kind, componentType component.Type) component.Factory {
	return s.factory
}

func (s *otelInput) GetExtensions() map[component.ID]component.Component {
	return map[component.ID]component.Component{}
}

func (s *otelInput) GetExporters() map[component.DataType]map[component.ID]component.Component {
	return map[component.DataType]map[component.ID]component.Component{}
}

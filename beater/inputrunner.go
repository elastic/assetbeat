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

package beater

import (
	"flag"
	"fmt"
	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	"strings"
	"sync"
	"time"

	"github.com/elastic/beats/v7/libbeat/autodiscover"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/cfgfile"
	"github.com/elastic/beats/v7/libbeat/common/reload"
	"github.com/elastic/beats/v7/libbeat/management"
	"github.com/elastic/beats/v7/libbeat/publisher/pipetool"
	"github.com/elastic/beats/v7/libbeat/statestore"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/monitoring"
	"github.com/elastic/go-concert/unison"
	"github.com/elastic/inputrunner/channel"
	cfg "github.com/elastic/inputrunner/config"

	"github.com/elastic/beats/v7/filebeat/input/v2/compat"
)

var once = flag.Bool("once", false, "Run inputrunner only once until all harvesters reach EOF")

// Inputrunner is a beater object. Contains all objects needed to run the beat
type Inputrunner struct {
	config        *cfg.Config
	pluginFactory PluginFactory
	done          chan struct{}
	stopOnce      sync.Once // wraps the Stop() method
	pipeline      beat.PipelineConnector
}

type PluginFactory func(beat.Info, *logp.Logger, StateStore) []v2.Plugin

type StateStore interface {
	Access() (*statestore.Store, error)
	CleanupInterval() time.Duration
}

// New creates a new Inputrunner pointer instance.
func New(plugins PluginFactory) beat.Creator {
	return func(b *beat.Beat, rawConfig *conf.C) (beat.Beater, error) {
		return newBeater(b, plugins, rawConfig)
	}
}

func newBeater(b *beat.Beat, plugins PluginFactory, rawConfig *conf.C) (beat.Beater, error) {
	config := cfg.DefaultConfig
	if err := rawConfig.Unpack(&config); err != nil {
		return nil, fmt.Errorf("Error reading config file: %w", err)
	}

	if err := config.FetchConfigs(); err != nil {
		return nil, err
	}

	/*if b.API != nil {
		if err = inputmon.AttachHandler(b.API.Router()); err != nil {
			return nil, fmt.Errorf("failed attach inputs api to monitoring endpoint server: %w", err)
		}
	}*/

	enabledInputs := config.ListEnabledInputs()
	var haveEnabledInputs bool
	if len(enabledInputs) > 0 {
		haveEnabledInputs = true
	}

	if !config.ConfigInput.Enabled() && !haveEnabledInputs && config.Autodiscover == nil && !b.Manager.Enabled() {
		if !b.InSetupCmd {
			return nil, fmt.Errorf("no inputs enabled and configuration reloading disabled. What inputs do you want me to run?")
		}

	}

	if config.IsInputEnabled("stdin") && len(enabledInputs) > 1 {
		return nil, fmt.Errorf("stdin requires to be run in exclusive mode, configured inputs: %s", strings.Join(enabledInputs, ", "))
	}

	ir := &Inputrunner{
		done:          make(chan struct{}),
		config:        &config,
		pluginFactory: plugins,
	}

	return ir, nil
}

// Run allows the beater to be run as a beat.
func (ir *Inputrunner) Run(b *beat.Beat) error {
	var err error
	config := ir.config

	waitFinished := newSignalWait()
	waitEvents := newSignalWait()

	// count active events for waiting on shutdown
	wgEvents := &eventCounter{
		count: monitoring.NewInt(nil, "filebeat.events.active"), // Gauge
		added: monitoring.NewUint(nil, "filebeat.events.added"),
		done:  monitoring.NewUint(nil, "filebeat.events.done"),
	}
	//finishedLogger := newFinishedLogger(wgEvents)

	// setup event counting for startup and a global common ACKer, such that all events will be
	// routed to the reigstrar after they've been ACKed.
	// Events with Private==nil or the type of private != file.State are directly
	// forwarded to `finishedLogger`. Events from the `logs` input will first be forwarded
	// to the registrar via `registrarChannel`, which finally forwards the events to finishedLogger as well.
	// The finishedLogger decrements the counters in wgEvents after all events have been securely processed
	// by the registry.
	ir.pipeline = withPipelineEventCounter(b.Publisher, wgEvents)
	//ir.pipeline = pipetool.WithACKer(ir.pipeline, eventACKer(finishedLogger, registrarChannel))

	// Inputrunner by default required infinite retry. Let's configure this for all
	// inputs by default.  Inputs (and InputController) can overwrite the sending
	// guarantees explicitly when connecting with the pipeline.
	ir.pipeline = pipetool.WithDefaultGuarantees(ir.pipeline, beat.GuaranteedSend)

	outDone := make(chan struct{}) // outDone closes down all active pipeline connections

	inputsLogger := logp.NewLogger("input")
	v2Inputs := ir.pluginFactory(b.Info, inputsLogger, nil)
	v2InputLoader, err := v2.NewLoader(inputsLogger, v2Inputs, "type", cfg.DefaultType)
	if err != nil {
		panic(err) // loader detected invalid state.
	}

	var inputTaskGroup unison.TaskGroup
	defer func() {
		_ = inputTaskGroup.Stop()
	}()
	if err := v2InputLoader.Init(&inputTaskGroup, v2.ModeRun); err != nil {
		logp.Err("Failed to initialize the input managers: %v", err)
		return err
	}

	inputLoader := channel.RunnerFactoryWithCommonInputSettings(b.Info,
		compat.RunnerFactory(inputsLogger, b.Info, v2InputLoader),
	)

	crawler, err := newCrawler(inputLoader, nil, config.Inputs, ir.done, *once)
	if err != nil {
		logp.Err("Could not init crawler: %v", err)
		return err
	}

	// The order of starting and stopping is important. Stopping is inverted to the starting order.
	// The current order is: registrar, publisher, spooler, crawler
	// That means, crawler is stopped first.

	// Stopping publisher (might potentially drop items)
	defer func() {
		// Closes first the registrar logger to make sure not more events arrive at the registrar
		// registrarChannel must be closed first to potentially unblock (pretty unlikely) the publisher
		//registrarChannel.Close()
		close(outDone) // finally close all active connections to publisher pipeline
	}()

	// Wait for all events to be processed or timeout
	defer waitEvents.Wait()

	err = crawler.Start(ir.pipeline, config.ConfigInput)
	if err != nil {
		crawler.Stop()
		return fmt.Errorf("Failed to start crawler: %w", err)
	}

	// If run once, add crawler completion check as alternative to done signal
	if *once {
		runOnce := func() {
			logp.Info("Running filebeat once. Waiting for completion ...")
			crawler.WaitForCompletion()
			logp.Info("All data collection completed. Shutting down.")
		}
		waitFinished.Add(runOnce)
	}

	// Register reloadable list of inputs and modules
	inputs := cfgfile.NewRunnerList(management.DebugK, inputLoader, ir.pipeline)
	reload.RegisterV2.MustRegisterInput(inputs)

	modules := cfgfile.NewRunnerList(management.DebugK, nil, ir.pipeline)

	var adiscover *autodiscover.Autodiscover
	if ir.config.Autodiscover != nil {
		adiscover, err = autodiscover.NewAutodiscover(
			"inputrunner",
			ir.pipeline,
			cfgfile.MultiplexedRunnerFactory(
				cfgfile.MatchDefault(inputLoader),
			),
			autodiscover.QueryConfig(),
			config.Autodiscover,
			b.Keystore,
		)
		if err != nil {
			return err
		}
	}
	adiscover.Start()

	// We start the manager when all the subsystem are initialized and ready to received events.
	if err := b.Manager.Start(); err != nil {
		return err
	}

	// Add done channel to wait for shutdown signal
	waitFinished.AddChan(ir.done)
	waitFinished.Wait()

	// Stop reloadable lists, autodiscover -> Stop crawler -> stop inputs -> stop harvesters
	// Note: waiting for crawlers to stop here in order to install wgEvents.Wait
	//       after all events have been enqueued for publishing. Otherwise wgEvents.Wait
	//       or publisher might panic due to concurrent updates.
	inputs.Stop()
	modules.Stop()
	adiscover.Stop()
	crawler.Stop()

	timeout := ir.config.ShutdownTimeout
	// Checks if on shutdown it should wait for all events to be published
	waitPublished := ir.config.ShutdownTimeout > 0 || *once
	if waitPublished {
		// Wait for registrar to finish writing registry
		waitEvents.Add(withLog(wgEvents.Wait,
			"Continue shutdown: All enqueued events being published."))
		// Wait for either timeout or all events having been ACKed by outputs.
		if ir.config.ShutdownTimeout > 0 {
			logp.Info("Shutdown output timer started. Waiting for max %v.", timeout)
			waitEvents.Add(withLog(waitDuration(timeout),
				"Continue shutdown: Time out waiting for events being published."))
		} else {
			waitEvents.AddChan(ir.done)
		}
	}

	// Stop the manager and stop the connection to any dependent services.
	b.Manager.Stop()

	return nil
}

// Stop is called on exit to stop the crawling, spooling and registration processes.
func (ir *Inputrunner) Stop() {
	logp.Info("Stopping inputrunner")

	ir.stopOnce.Do(func() { close(ir.done) })
}

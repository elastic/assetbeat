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
	"sync"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/publisher/pipetool"
	"github.com/elastic/elastic-agent-libs/monitoring"
)

type eventCounter struct {
	added *monitoring.Uint
	done  *monitoring.Uint
	count *monitoring.Int
	wg    sync.WaitGroup
}

// countingClient adds and subtracts from a counter when events have been
// published, dropped or ACKed. The countingClient can be used to keep track of
// inflight events for a beat.Client instance. The counter is updated after the
// client has been disconnected from the publisher pipeline via 'Closed'.
type countingClient struct {
	counter *eventCounter
	client  beat.Client
}

type countingEventer struct {
	wgEvents *eventCounter
}

type combinedEventer struct {
	a, b beat.ClientEventer
}

func (c *eventCounter) Add(delta int) {
	c.count.Add(int64(delta))
	c.added.Add(uint64(delta))
	c.wg.Add(delta)
}

func (c *eventCounter) Done() {
	c.count.Dec()
	c.done.Inc()
	c.wg.Done()
}

func (c *eventCounter) Wait() {
	c.wg.Wait()
}

// withPipelineEventCounter adds a counter to the pipeline that keeps track of
// all events published, dropped and ACKed by any active client.
// The type accepted by counter is compatible with sync.WaitGroup.
func withPipelineEventCounter(pipeline beat.PipelineConnector, counter *eventCounter) beat.PipelineConnector {
	counterListener := &countingEventer{counter}

	pipeline = pipetool.WithClientConfigEdit(pipeline, func(config beat.ClientConfig) (beat.ClientConfig, error) {
		if evts := config.Events; evts != nil {
			config.Events = &combinedEventer{evts, counterListener}
		} else {
			config.Events = counterListener
		}
		return config, nil
	})

	pipeline = pipetool.WithClientWrapper(pipeline, func(client beat.Client) beat.Client {
		return &countingClient{
			counter: counter,
			client:  client,
		}
	})
	return pipeline
}

func (c *countingClient) Publish(event beat.Event) {
	c.counter.Add(1)
	c.client.Publish(event)
}

func (c *countingClient) PublishAll(events []beat.Event) {
	c.counter.Add(len(events))
	c.client.PublishAll(events)
}

func (c *countingClient) Close() error {
	return c.client.Close()
}

func (*countingEventer) Closing()   {}
func (*countingEventer) Closed()    {}
func (*countingEventer) Published() {}

func (c *countingEventer) FilteredOut(_ beat.Event) {}
func (c *countingEventer) DroppedOnPublish(_ beat.Event) {
	c.wgEvents.Done()
}

func (c *combinedEventer) Closing() {
	c.a.Closing()
	c.b.Closing()
}

func (c *combinedEventer) Closed() {
	c.a.Closed()
	c.b.Closed()
}

func (c *combinedEventer) Published() {
	c.a.Published()
	c.b.Published()
}

func (c *combinedEventer) FilteredOut(event beat.Event) {
	c.a.FilteredOut(event)
	c.b.FilteredOut(event)
}

func (c *combinedEventer) DroppedOnPublish(event beat.Event) {
	c.a.DroppedOnPublish(event)
	c.b.DroppedOnPublish(event)
}

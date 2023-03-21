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
	"sync"
	"testing"
	"time"

	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/inputrunner/input/testutil"
	v2 "github.com/elastic/inputrunner/input/v2"
	"github.com/stretchr/testify/assert"
)

func TestPlugin(t *testing.T) {
	p := Plugin()
	assert.Equal(t, "otel", p.Name)
	assert.NotNil(t, p.Manager)
}

func TestOtelInput_RunDockerStats(t *testing.T) {
	publisher := testutil.NewInMemoryPublisher()

	ctx, cancel := context.WithCancel(context.Background())
	inputCtx := v2.Context{
		Logger:      logp.NewLogger("test"),
		Cancelation: ctx,
	}

	cfg := defaultConfig()
	cfg.ReceiverName = "docker_stats"

	input, err := newInput(cfg)
	assert.NoError(t, err)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		err = input.Run(inputCtx, publisher)
		assert.NoError(t, err)
	}()

	time.Sleep(20 * time.Second)
	cancel()
	timeout := time.After(time.Second)
	closeCh := make(chan struct{})
	go func() {
		defer close(closeCh)
		wg.Wait()
	}()
	select {
	case <-timeout:
		t.Fatal("Test timed out")
	case <-closeCh:
		// Waitgroup finished in time, nothing to do
	}
}

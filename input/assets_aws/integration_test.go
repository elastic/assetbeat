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

//go:build integration
// +build integration

package assets_aws

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/elastic/elastic-agent-libs/logp"
	input "github.com/elastic/inputrunner/input/v2"
	"github.com/elastic/inputrunner/mocks"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestRun(t *testing.T) {
	ctrl := gomock.NewController(t)
	publisher := mocks.NewMockPublisher(ctrl)

	ctx, cancel := context.WithCancel(context.Background())
	inputCtx := input.Context{
		Logger:      logp.NewLogger("test"),
		Cancelation: ctx,
	}

	runner, err := newAssetsAWS(defaultConfig())
	assert.NoError(t, err)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		err = runner.Run(inputCtx, publisher)
		assert.NoError(t, err)
	}()

	time.Sleep(time.Millisecond)
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

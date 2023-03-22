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

// Package compat provides helpers for integrating the input/v2 API with
// existing input based features like autodiscovery, config file reloading, or
// filebeat modules.
package context

import (
	"context"
	"testing"

	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/stretchr/testify/assert"
)

func TestContextLogger(t *testing.T) {
	ctx := context.Background()

	assert.Equal(t, logp.NewLogger("empty"), Logger(ctx))

	logger := logp.NewLogger("empty")
	ctx = WithLogger(ctx, logger)
	assert.Equal(t, logger, Logger(ctx))
}

func TestContextID(t *testing.T) {
	ctx := context.Background()

	assert.Equal(t, "", ID(ctx))
	ctx = WithID(ctx, "test")
	assert.Equal(t, "test", ID(ctx))
}

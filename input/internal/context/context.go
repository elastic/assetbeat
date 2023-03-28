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

package context

import (
	"context"

	"github.com/elastic/elastic-agent-libs/logp"
)

type contextKeyType int

const (
	IDKey contextKeyType = iota
	loggerKey
)

// WithLogger returns a copy of the context with a logger assigned
func WithLogger(ctx context.Context, logger *logp.Logger) context.Context {
	return context.WithValue(ctx, loggerKey, logger)
}

// Logger returns the logger assigned to the context
func Logger(ctx context.Context) *logp.Logger {
	if l, ok := ctx.Value(loggerKey).(*logp.Logger); ok {
		return l
	}
	return logp.NewLogger("empty")
}

// WithID returns a copy of the context with the specified input ID stored
func WithID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, IDKey, id)
}

// ID returns the input ID assigned to the context
func ID(ctx context.Context) string {
	if i, ok := ctx.Value(IDKey).(string); ok {
		return i
	}

	return ""
}

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

package testutil

import (
	"sync"

	"github.com/elastic/beats/v7/libbeat/beat"
)

// InMemoryPublisher is a publisher which stores events in memory, to be used
// in unit tests
type InMemoryPublisher struct {
	mu     sync.Mutex
	Events []beat.Event
}

// NewInMemoryPublisher creates a new instance of InMemoryPublisher
func NewInMemoryPublisher() *InMemoryPublisher {
	return &InMemoryPublisher{}
}

// Publish stores a new event in memory
func (p *InMemoryPublisher) Publish(e beat.Event) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.Events = append(p.Events, e)
}

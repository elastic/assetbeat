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

package internal

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/assetbeat/input/testutil"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func TestECS_WithCloudInstanceId(t *testing.T) {
	for _, tt := range []struct {
		name     string
		assetOp  AssetOption
		expected beat.Event
	}{
		{
			name:    "instance Id provided",
			assetOp: WithCloudInstanceId("i-0699b78f46f0fa248"),
			expected: beat.Event{
				Fields: mapstr.M{"cloud.instance.id": "i-0699b78f46f0fa248"},
				Meta:   mapstr.M{},
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			publisher := testutil.NewInMemoryPublisher()

			Publish(publisher, nil, tt.assetOp)

			assert.Equal(t, 1, len(publisher.Events))
			assert.Equal(t, tt.expected, publisher.Events[0])
		})
	}
}

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

func TestIndex_WithIndex(t *testing.T) {
	for _, tt := range []struct {
		name     string
		assetOp  AssetOption
		expected beat.Event
	}{
		{
			name:    "no defined index namespace, the default is used",
			assetOp: WithIndex("aws.ec2.instance", ""),
			expected: beat.Event{
				Fields: mapstr.M{},
				Meta:   mapstr.M{"index": "assets-aws.ec2.instance-default"},
			},
		},
		{
			name:    "defined namespace",
			assetOp: WithIndex("aws.ec2.instance", "test.namespace"),
			expected: beat.Event{
				Fields: mapstr.M{},
				Meta:   mapstr.M{"index": "assets-aws.ec2.instance-test.namespace"},
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			publisher := testutil.NewInMemoryPublisher()

			Publish(publisher, tt.assetOp)

			assert.Equal(t, 1, len(publisher.Events))
			assert.Equal(t, tt.expected, publisher.Events[0])
		})
	}
}

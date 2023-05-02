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

package aws

import (
	"testing"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/inputrunner/input/assets/internal"
	"github.com/elastic/inputrunner/input/testutil"
	"github.com/stretchr/testify/assert"
)

func TestWithAssetTags(t *testing.T) {
	for _, tt := range []struct {
		name string

		opts          []internal.AssetOption
		expectedEvent beat.Event
	}{
		{
			name: "with valid tags",
			opts: []internal.AssetOption{
				internal.WithAssetCloudProvider("aws"),
				WithAssetTags(mapstr.M{"tag1": "a", "tag2": "b"}),
			},
			expectedEvent: beat.Event{Fields: mapstr.M{
				"cloud.provider":           "aws",
				"asset.metadata.tags.tag1": "a",
				"asset.metadata.tags.tag2": "b",
			}, Meta: mapstr.M{}},
		},
		{
			name: "with valid tags and metadata",
			opts: []internal.AssetOption{
				internal.WithAssetCloudProvider("aws"),
				internal.WithAssetMetadata(mapstr.M{"foo": "bar"}),
				WithAssetTags(mapstr.M{"tag1": "a", "tag2": "b"}),
			},
			expectedEvent: beat.Event{Fields: mapstr.M{
				"cloud.provider":           "aws",
				"asset.metadata.foo":       "bar",
				"asset.metadata.tags.tag1": "a",
				"asset.metadata.tags.tag2": "b",
			}, Meta: mapstr.M{}},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			publisher := testutil.NewInMemoryPublisher()

			internal.Publish(publisher, tt.opts...)

			assert.Equal(t, 1, len(publisher.Events))
			assert.Equal(t, tt.expectedEvent, publisher.Events[0])
		})
	}
}

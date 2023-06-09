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

package gcp

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/assetbeat/input/internal"
	"github.com/elastic/assetbeat/input/testutil"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func TestWithAssetLabels(t *testing.T) {
	for _, tt := range []struct {
		name string

		opts          []internal.AssetOption
		expectedEvent beat.Event
	}{
		{
			name: "with valid labels",
			opts: []internal.AssetOption{
				internal.WithAssetCloudProvider("gcp"),
				WithAssetLabels(mapstr.M{"label1": "a", "label2": "b"}),
			},
			expectedEvent: beat.Event{Fields: mapstr.M{
				"cloud.provider":               "gcp",
				"asset.metadata.labels.label1": "a",
				"asset.metadata.labels.label2": "b",
			}, Meta: mapstr.M{}},
		},
		{
			name: "with valid labels and metadata",
			opts: []internal.AssetOption{
				internal.WithAssetCloudProvider("gcp"),
				internal.WithAssetMetadata(mapstr.M{"foo": "bar"}),
				WithAssetLabels(mapstr.M{"label1": "a", "label2": "b"}),
			},
			expectedEvent: beat.Event{Fields: mapstr.M{
				"cloud.provider":               "gcp",
				"asset.metadata.foo":           "bar",
				"asset.metadata.labels.label1": "a",
				"asset.metadata.labels.label2": "b",
			}, Meta: mapstr.M{}},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			publisher := testutil.NewInMemoryPublisher()

			internal.Publish(publisher, nil, tt.opts...)

			assert.Equal(t, 1, len(publisher.Events))
			assert.Equal(t, tt.expectedEvent, publisher.Events[0])
		})
	}
}

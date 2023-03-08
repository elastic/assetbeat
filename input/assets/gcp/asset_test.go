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

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/inputrunner/input/assets/internal"
	"github.com/elastic/inputrunner/mocks"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestWithAssetLabels(t *testing.T) {
	for _, tt := range []struct {
		name string

		opts          []internal.AssetOption
		expectedEvent beat.Event
		expectedError error
	}{
		{
			name: "with valid labels",
			opts: []internal.AssetOption{
				internal.WithAssetCloudProvider("aws"),
				WithAssetLabels(map[string]string{"label1": "a", "label2": "b"}),
			},
			expectedEvent: beat.Event{Fields: mapstr.M{
				"cloud.provider": "aws",
				"asset.metadata": mapstr.M{
					"labels": map[string]string{"label1": "a", "label2": "b"},
				},
			}},
		},
		{
			name: "with valid labels and metadata",
			opts: []internal.AssetOption{
				internal.WithAssetCloudProvider("aws"),
				internal.WithAssetMetadata(mapstr.M{"foo": "bar"}),
				WithAssetLabels(map[string]string{"label1": "a", "label2": "b"}),
			},
			expectedEvent: beat.Event{Fields: mapstr.M{
				"cloud.provider": "aws",
				"asset.metadata": mapstr.M{
					"labels": map[string]string{"label1": "a", "label2": "b"},
					"foo":    "bar",
				},
			}},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			publisher := mocks.NewMockPublisher(ctrl)

			if tt.expectedError == nil {
				publisher.EXPECT().Publish(tt.expectedEvent)
			}

			err := internal.Publish(publisher, tt.opts...)

			if tt.expectedError != nil {
				assert.Equal(t, tt.expectedError, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

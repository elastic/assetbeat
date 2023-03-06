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
	"github.com/elastic/inputrunner/mocks"
	"github.com/golang/mock/gomock"
)

func TestPublishAsset(t *testing.T) {
	for _, tt := range []struct {
		name  string
		event beat.Event

		region    string
		assetType string
		assetID   string
		parents   []string
		children  []string
		metadata  mapstr.M
	}{
		{
			name: "required fields",
			event: beat.Event{
				Fields: mapstr.M{
					"cloud.provider": "gcp",
					"cloud.region":   "europe-central2-a",
					"asset.type":     "gcp.compute.instance",
					"asset.id":       "42",
					"asset.ean":      "gcp.compute.instance:42",
				},
			},

			region:    "europe-central2-a",
			assetType: "gcp.compute.instance",
			assetID:   "42",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			publisher := mocks.NewMockPublisher(ctrl)

			publisher.EXPECT().Publish(tt.event)
			publishAsset(
				publisher,
				tt.region,
				tt.assetType,
				tt.assetID,
				tt.parents,
				tt.children,
				tt.metadata,
			)
		})
	}
}

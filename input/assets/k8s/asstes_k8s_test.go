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

package k8s

import (
	"testing"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/inputrunner/mocks"
	"github.com/golang/mock/gomock"
)

func TestPublishK8sAsset(t *testing.T) {
	for _, tt := range []struct {
		name  string
		event beat.Event

		assetName        string
		assetType        string
		assetID          string
		parents          []string
		children         []string
		assetSpecificMap map[string]interface{}
	}{
		{
			name: "publish pod",
			event: beat.Event{
				Fields: mapstr.M{
					"asset.name":                "foo",
					"asset.type":                "k8s.pod",
					"asset.id":                  "a375d24b-fa20-4ea6-a0ee-1d38671d2c09",
					"asset.ean":                 "k8s.pod.foo",
					"kubernetes.pod.name":       "foo",
					"kubernetes.pod.uid":        "a375d24b-fa20-4ea6-a0ee-1d38671d2c09",
					"kubernetes.pod.start_time": "2023-02-14T17:38:53+02:00",
					"kubernetes.namespace":      "default",
				},
			},

			assetName: "foo",
			assetType: "k8s.pod",
			assetID:   "a375d24b-fa20-4ea6-a0ee-1d38671d2c09",
			assetSpecificMap: map[string]interface{}{
				"kubernetes.pod.name":       "foo",
				"kubernetes.pod.uid":        "a375d24b-fa20-4ea6-a0ee-1d38671d2c09",
				"kubernetes.pod.start_time": "2023-02-14T17:38:53+02:00",
				"kubernetes.namespace":      "default",
			},
		},
		{
			name: "publish node",
			event: beat.Event{
				Fields: mapstr.M{
					"asset.name":                 "ip-172-31-29-242.us-east-2.compute.internal",
					"asset.type":                 "k8s.node",
					"asset.id":                   "60988eed-1885-4b63-9fa4-780206969deb",
					"asset.ean":                  "k8s.node.ip-172-31-29-242.us-east-2.compute.internal",
					"kubernetes.node.name":       "ip-172-31-29-242.us-east-2.compute.internal",
					"kubernetes.node.providerId": "aws:///us-east-2b/i-0699b78f46f0fa248",
					"kubernetes.node.start_time": "2023-02-14T17:38:53+02:00",
				},
			},

			assetName: "ip-172-31-29-242.us-east-2.compute.internal",
			assetType: "k8s.node",
			assetID:   "60988eed-1885-4b63-9fa4-780206969deb",
			assetSpecificMap: map[string]interface{}{
				"kubernetes.node.name":       "ip-172-31-29-242.us-east-2.compute.internal",
				"kubernetes.node.providerId": "aws:///us-east-2b/i-0699b78f46f0fa248",
				"kubernetes.node.start_time": "2023-02-14T17:38:53+02:00",
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			publisher := mocks.NewMockPublisher(ctrl)

			publisher.EXPECT().Publish(tt.event)
			publishK8sAsset(
				tt.assetName,
				tt.assetType,
				tt.assetID,
				tt.parents,
				tt.children,
				publisher,
				tt.assetSpecificMap,
			)
		})
	}
}

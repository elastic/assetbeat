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
	"time"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/inputrunner/input/testutil"
)

var startTime = metav1.Time{Time: time.Date(2021, 8, 15, 14, 30, 45, 100, time.Local)}

func TestPublish(t *testing.T) {
	for _, tt := range []struct {
		name string

		opts          []AssetOption
		expectedEvent beat.Event
	}{
		{
			name:          "with no options",
			expectedEvent: beat.Event{Fields: mapstr.M{}, Meta: mapstr.M{}},
		},
		{
			name: "with a valid cloud provider name",
			opts: []AssetOption{
				WithAssetCloudProvider("aws"),
			},
			expectedEvent: beat.Event{Fields: mapstr.M{
				"cloud.provider": "aws",
			},
				Meta: mapstr.M{},
			},
		},
		{
			name: "with a valid region",
			opts: []AssetOption{
				WithAssetCloudProvider("aws"),
				WithAssetRegion("us-east-1"),
			},
			expectedEvent: beat.Event{Fields: mapstr.M{
				"cloud.provider": "aws",
				"cloud.region":   "us-east-1",
			},
				Meta: mapstr.M{},
			},
		},
		{
			name: "with a valid account ID",
			opts: []AssetOption{
				WithAssetCloudProvider("aws"),
				WithAssetAccountID("42"),
			},
			expectedEvent: beat.Event{Fields: mapstr.M{
				"cloud.provider":   "aws",
				"cloud.account.id": "42",
			},
				Meta: mapstr.M{},
			},
		},
		{
			name: "with a valid asset type and ID",
			opts: []AssetOption{
				WithAssetCloudProvider("aws"),
				WithAssetTypeAndID("aws.ec2.instance", "i-1234"),
			},
			expectedEvent: beat.Event{Fields: mapstr.M{
				"cloud.provider": "aws",
				"asset.type":     "aws.ec2.instance",
				"asset.id":       "i-1234",
				"asset.ean":      "aws.ec2.instance:i-1234",
			}, Meta: mapstr.M{},
			},
		},
		{
			name: "with valid parents",
			opts: []AssetOption{
				WithAssetCloudProvider("aws"),
				WithAssetParents([]string{"5678"}),
			},
			expectedEvent: beat.Event{Fields: mapstr.M{
				"cloud.provider": "aws",
				"asset.parents":  []string{"5678"},
			}, Meta: mapstr.M{}},
		},
		{
			name: "with valid children",
			opts: []AssetOption{
				WithAssetCloudProvider("aws"),
				WithAssetChildren([]string{"5678"}),
			},
			expectedEvent: beat.Event{Fields: mapstr.M{
				"cloud.provider": "aws",
				"asset.children": []string{"5678"},
			}, Meta: mapstr.M{}},
		},
		{
			name: "with valid metadata",
			opts: []AssetOption{
				WithAssetCloudProvider("aws"),
				WithAssetMetadata(mapstr.M{"foo": "bar"}),
			},
			expectedEvent: beat.Event{Fields: mapstr.M{
				"cloud.provider":     "aws",
				"asset.metadata.foo": "bar",
			}, Meta: mapstr.M{}},
		},
		{
			name: "with valid node data",
			opts: []AssetOption{
				WithNodeData("ip-172-31-29-242.us-east-2.compute.internal", "aws:///us-east-2b/i-0699b78f46f0fa248", &startTime),
			},
			expectedEvent: beat.Event{Fields: mapstr.M{
				"kubernetes.node.name":       "ip-172-31-29-242.us-east-2.compute.internal",
				"kubernetes.node.providerId": "aws:///us-east-2b/i-0699b78f46f0fa248",
				"kubernetes.node.start_time": &startTime,
			}, Meta: mapstr.M{}},
		},
		{
			name: "with valid pod data",
			opts: []AssetOption{
				WithPodData("nginx", "a375d24b-fa20-4ea6-a0ee-1d38671d2c09", "default", &startTime),
			},
			expectedEvent: beat.Event{Fields: mapstr.M{
				"kubernetes.pod.name":       "nginx",
				"kubernetes.pod.uid":        "a375d24b-fa20-4ea6-a0ee-1d38671d2c09",
				"kubernetes.pod.start_time": &startTime,
				"kubernetes.namespace":      "default",
			}, Meta: mapstr.M{}},
		}, {
			name: "with valid container data",
			opts: []AssetOption{
				WithContainerData("nginx-container", "a375d24b-fa20-4ea6-a0ee-1d38671d2c09", "default", "running", &startTime),
			},
			expectedEvent: beat.Event{Fields: mapstr.M{
				"kubernetes.container.name":       "nginx-container",
				"kubernetes.container.uid":        "a375d24b-fa20-4ea6-a0ee-1d38671d2c09",
				"kubernetes.container.start_time": &startTime,
				"kubernetes.container.state":      "running",
				"kubernetes.namespace":            "default",
			}, Meta: mapstr.M{}},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			publisher := testutil.NewInMemoryPublisher()

			Publish(publisher, tt.opts...)
			assert.Equal(t, 1, len(publisher.Events))
			assert.Equal(t, tt.expectedEvent, publisher.Events[0])
		})
	}
}

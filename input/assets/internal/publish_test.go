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
	"errors"
	"testing"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/inputrunner/mocks"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestPublish(t *testing.T) {
	for _, tt := range []struct {
		name string

		opts          []AssetOption
		expectedEvent beat.Event
		expectedError error
	}{
		{
			name:          "with no options",
			expectedError: errors.New("a cloud provider name is required"),
		},
		{
			name: "with an empty cloud provider name",
			opts: []AssetOption{
				WithAssetCloudProvider(""),
			},
			expectedError: errors.New("a cloud provider name is required"),
		},
		{
			name: "with a valid cloud provider name",
			opts: []AssetOption{
				WithAssetCloudProvider("aws"),
			},
			expectedEvent: beat.Event{Fields: mapstr.M{
				"cloud.provider": "aws",
			}},
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
			}},
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
			}},
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
			}},
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
			}},
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
			}},
		},
		{
			name: "with valid metadata",
			opts: []AssetOption{
				WithAssetCloudProvider("aws"),
				WithAssetMetadata(mapstr.M{"foo": "bar"}),
			},
			expectedEvent: beat.Event{Fields: mapstr.M{
				"cloud.provider": "aws",
				"asset.metadata": mapstr.M{"foo": "bar"},
			}},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			publisher := mocks.NewMockPublisher(ctrl)

			if tt.expectedError == nil {
				publisher.EXPECT().Publish(tt.expectedEvent)
			}

			err := Publish(publisher, tt.opts...)

			if tt.expectedError != nil {
				assert.Equal(t, tt.expectedError, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

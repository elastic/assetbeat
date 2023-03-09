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

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/inputrunner/mocks"
	"github.com/golang/mock/gomock"
)

func TestPublish(t *testing.T) {
	for _, tt := range []struct {
		name string

		opts          []AssetOption
		expectedEvent beat.Event
	}{
		{
			name: "with no options",
			expectedEvent: beat.Event{Fields: mapstr.M{
				"cloud.provider": "aws",
				"asset.type":     "aws.ec2.instance",
				"asset.id":       "i-1234",
				"asset.ean":      "aws.ec2.instance:i-1234",
			}},
		},
		{
			name: "with a valid region",
			opts: []AssetOption{
				WithAssetRegion("us-east-1"),
			},
			expectedEvent: beat.Event{Fields: mapstr.M{
				"cloud.provider": "aws",
				"asset.type":     "aws.ec2.instance",
				"asset.id":       "i-1234",
				"asset.ean":      "aws.ec2.instance:i-1234",
				"cloud.region":   "us-east-1",
			}},
		},
		{
			name: "with a valid account ID",
			opts: []AssetOption{
				WithAssetAccountID("42"),
			},
			expectedEvent: beat.Event{Fields: mapstr.M{
				"cloud.provider":   "aws",
				"asset.type":       "aws.ec2.instance",
				"asset.id":         "i-1234",
				"asset.ean":        "aws.ec2.instance:i-1234",
				"cloud.account.id": "42",
			}},
		},
		{
			name: "with valid parents",
			opts: []AssetOption{
				WithAssetParents([]string{"5678"}),
			},
			expectedEvent: beat.Event{Fields: mapstr.M{
				"cloud.provider": "aws",
				"asset.type":     "aws.ec2.instance",
				"asset.id":       "i-1234",
				"asset.ean":      "aws.ec2.instance:i-1234",
				"asset.parents":  []string{"5678"},
			}},
		},
		{
			name: "with valid children",
			opts: []AssetOption{
				WithAssetChildren([]string{"5678"}),
			},
			expectedEvent: beat.Event{Fields: mapstr.M{
				"cloud.provider": "aws",
				"asset.type":     "aws.ec2.instance",
				"asset.id":       "i-1234",
				"asset.ean":      "aws.ec2.instance:i-1234",
				"asset.children": []string{"5678"},
			}},
		},
		{
			name: "with valid metadata",
			opts: []AssetOption{
				WithAssetMetadata(mapstr.M{"foo": "bar"}),
			},
			expectedEvent: beat.Event{Fields: mapstr.M{
				"cloud.provider": "aws",
				"asset.type":     "aws.ec2.instance",
				"asset.id":       "i-1234",
				"asset.ean":      "aws.ec2.instance:i-1234",
				"asset.metadata": mapstr.M{"foo": "bar"},
			}},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			publisher := mocks.NewMockPublisher(ctrl)

			publisher.EXPECT().Publish(tt.expectedEvent)

			req := AssetRequiredFields{
				CloudProvider: "aws",
				AssetType:     "aws.ec2.instance",
				AssetID:       "i-1234",
			}

			Publish(publisher, req, tt.opts...)
		})
	}
}

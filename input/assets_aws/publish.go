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

package assets_aws

import (
	"fmt"

	stateless "github.com/elastic/inputrunner/input/v2/input-stateless"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func publishAWSAsset(publisher stateless.Publisher, region, account, assetType, assetID string, parents, children []string, tags map[string]string, metadata mapstr.M) {
	asset := mapstr.M{
		"cloud.provider":   "aws",
		"cloud.region":     region,
		"cloud.account.id": account,

		"asset.type": assetType,
		"asset.id":   assetID,
		"asset.ean":  fmt.Sprintf("%s:%s", assetType, assetID),
	}

	if parents != nil {
		asset["asset.parents"] = parents
	}

	if children != nil {
		asset["asset.children"] = children
	}

	assetMetadata := mapstr.M{}
	if tags != nil {
		assetMetadata["tags"] = tags
	}
	assetMetadata.Update(metadata)
	if len(assetMetadata) != 0 {
		asset["asset.metadata"] = assetMetadata
	}

	publisher.Publish(beat.Event{Fields: asset})
}

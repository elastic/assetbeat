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
	"fmt"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/elastic-agent-libs/mapstr"
	stateless "github.com/elastic/inputrunner/input/v2/input-stateless"
)

type AssetOption func(beat.Event) beat.Event

// Publish emits a `beat.Event` to the specified publisher, with the provided
// parameters
func Publish(publisher stateless.Publisher, req AssetRequiredFields, opts ...AssetOption) {
	event := beat.Event{Fields: mapstr.M{
		"cloud.provider": req.CloudProvider,
		"asset.type":     req.AssetType,
		"asset.id":       req.AssetID,
		"asset.ean":      fmt.Sprintf("%s:%s", req.AssetType, req.AssetID),
	}}

	for _, o := range opts {
		event = o(event)
	}

	publisher.Publish(event)
}

type AssetRequiredFields struct {
	CloudProvider string
	AssetType     string
	AssetID       string
}

func WithAssetRegion(value string) AssetOption {
	return func(e beat.Event) beat.Event {
		e.Fields["cloud.region"] = value
		return e
	}
}

func WithAssetAccountID(value string) AssetOption {
	return func(e beat.Event) beat.Event {
		e.Fields["cloud.account.id"] = value
		return e
	}
}

func WithAssetParents(value []string) AssetOption {
	return func(e beat.Event) beat.Event {
		e.Fields["asset.parents"] = value
		return e
	}
}

func WithAssetChildren(value []string) AssetOption {
	return func(e beat.Event) beat.Event {
		e.Fields["asset.children"] = value
		return e
	}
}

func WithAssetMetadata(value mapstr.M) AssetOption {
	return func(e beat.Event) beat.Event {
		m := mapstr.M{}
		if e.Fields["asset.metadata"] != nil {
			m = e.Fields["asset.metadata"].(mapstr.M)
		}

		m.Update(value)
		e.Fields["asset.metadata"] = m
		return e
	}
}

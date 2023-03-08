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
	"fmt"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/elastic-agent-libs/mapstr"
	stateless "github.com/elastic/inputrunner/input/v2/input-stateless"
)

type EventOption func(beat.Event) beat.Event

// Publish emits a `beat.Event` to the specified publisher, with the provided
// parameters
func Publish(publisher stateless.Publisher, opts ...EventOption) error {
	event := beat.Event{Fields: mapstr.M{}}

	for _, o := range opts {
		event = o(event)
	}

	if event.Fields["cloud.provider"] == nil || event.Fields["cloud.provider"] == "" {
		return errors.New("a cloud provider name is required")
	}

	if event.Fields["asset.type"] != nil && event.Fields["asset.id"] != nil {
		event.Fields["asset.ean"] = fmt.Sprintf("%s:%s", event.Fields["asset.type"], event.Fields["asset.id"])
	}

	publisher.Publish(event)
	return nil
}

func WithEventCloudProvider(value string) EventOption {
	return func(e beat.Event) beat.Event {
		e.Fields["cloud.provider"] = value
		return e
	}
}

func WithEventRegion(value string) EventOption {
	return func(e beat.Event) beat.Event {
		e.Fields["cloud.region"] = value
		return e
	}
}

func WithEventAccountID(value string) EventOption {
	return func(e beat.Event) beat.Event {
		e.Fields["cloud.account.id"] = value
		return e
	}
}

func WithEventAssetType(value string) EventOption {
	return func(e beat.Event) beat.Event {
		e.Fields["asset.type"] = value
		return e
	}
}

func WithEventAssetID(value string) EventOption {
	return func(e beat.Event) beat.Event {
		e.Fields["asset.id"] = value
		return e
	}
}

func WithEventParents(value []string) EventOption {
	return func(e beat.Event) beat.Event {
		e.Fields["asset.parents"] = value
		return e
	}
}

func WithEventChildren(value []string) EventOption {
	return func(e beat.Event) beat.Event {
		e.Fields["asset.children"] = value
		return e
	}
}

func WithEventMetadata(value mapstr.M) EventOption {
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

func WithEventTags(value map[string]string) EventOption {
	return WithEventMetadata(mapstr.M{
		"tags": value,
	})
}

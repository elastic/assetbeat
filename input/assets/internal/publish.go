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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type AssetOption func(beat.Event) beat.Event

// Publish emits a `beat.Event` to the specified publisher, with the provided
// parameters
func Publish(publisher stateless.Publisher, opts ...AssetOption) {
	event := beat.Event{Fields: mapstr.M{}, Meta: mapstr.M{}}
	for _, o := range opts {
		event = o(event)
	}
	publisher.Publish(event)
}

func WithIndex(value string) AssetOption {
	return func(e beat.Event) beat.Event {
		e.Meta["index"] = fmt.Sprintf("%s-%s-%s", "assets", value, "default")
		return e
	}
}

func WithAssetCloudProvider(value string) AssetOption {
	return func(e beat.Event) beat.Event {
		e.Fields["cloud.provider"] = value
		return e
	}
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

func WithAssetTypeAndID(t, id string) AssetOption {
	return func(e beat.Event) beat.Event {
		e.Fields["asset.type"] = t
		e.Fields["asset.id"] = id
		e.Fields["asset.ean"] = fmt.Sprintf("%s:%s", t, id)
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

func WithNodeData(name, providerId string, startTime *metav1.Time) AssetOption {
	return func(e beat.Event) beat.Event {
		e.Fields["kubernetes.node.name"] = name
		e.Fields["kubernetes.node.providerId"] = providerId
		e.Fields["kubernetes.node.start_time"] = startTime
		return e
	}
}

func WithPodData(name, uid, namespace string, startTime *metav1.Time) AssetOption {
	return func(e beat.Event) beat.Event {
		e.Fields["kubernetes.pod.name"] = name
		e.Fields["kubernetes.pod.uid"] = uid
		e.Fields["kubernetes.pod.start_time"] = startTime
		e.Fields["kubernetes.namespace"] = namespace
		return e
	}
}

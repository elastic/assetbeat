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

	"github.com/stretchr/testify/assert"
)

func TestGetResourceNameFromURL(t *testing.T) {
	for _, tt := range []struct {
		name string

		URL          string
		expectedName string
	}{
		{
			name: "with an empty value",

			URL:          "",
			expectedName: "",
		},
		{
			name: "with a valid URL",

			URL:          "https://www.googleapis.com/compute/v1/projects/my_project/global/networks/my_network",
			expectedName: "my_network",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expectedName, getResourceNameFromURL(tt.URL))
		})
	}
}

func TestGetRegionFromZoneURL(t *testing.T) {
	for _, tt := range []struct {
		name string

		URL          string
		expectedName string
	}{
		{
			name: "with an empty value",

			URL:          "",
			expectedName: "",
		},
		{
			name: "with a valid URL",

			URL:          "https://www.googleapis.com/compute/v1/projects/my_project/zones/europe-west1-d",
			expectedName: "europe-west1",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expectedName, getRegionFromZoneURL(tt.URL))
		})
	}
}

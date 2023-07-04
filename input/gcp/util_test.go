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

	"cloud.google.com/go/container/apiv1/containerpb"

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

func TestGetVpcIdFromLink(t *testing.T) {
	vpcAssetsCache := getTestVpcCache()
	for _, tt := range []struct {
		name string

		selfLink   string
		expectedId string
	}{
		{
			name: "with a non existing selfLink",

			selfLink:   "https://www.googleapis.com/compute/v1/projects/my_project/global/networks/test",
			expectedId: "",
		},
		{
			name: "with an existing selfLink",

			selfLink:   "https://www.googleapis.com/compute/v1/projects/my_project/global/networks/my_network",
			expectedId: "1",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expectedId, getVpcIdFromLink(tt.selfLink, vpcAssetsCache))
		})
	}
}

func TestGetNetSelfLinkFromNetConfig(t *testing.T) {

	for _, tt := range []struct {
		name string

		networkConfig    *containerpb.NetworkConfig
		expectedSelfLink string
	}{
		{
			name: "with a non existing network",

			networkConfig: &containerpb.NetworkConfig{
				Network: "",
			},
			expectedSelfLink: "",
		},
		{
			name: "with an existing network",

			networkConfig: &containerpb.NetworkConfig{
				Network: "projects/my_project/global/networks/my_network",
			},
			expectedSelfLink: "https://www.googleapis.com/compute/v1/projects/my_project/global/networks/my_network",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expectedSelfLink, getNetSelfLinkFromNetConfig(tt.networkConfig))
		})
	}
}

func TestWantRegion(t *testing.T) {

	for _, tt := range []struct {
		name string

		confRegions    []string
		region         string
		expectedWanted bool
	}{
		{
			name: "wanted",

			confRegions:    []string{"us-east-1", "us-west-1"},
			region:         "us-east-1",
			expectedWanted: true,
		},
		{
			name: "not wanted",

			confRegions:    []string{"us-east-1", "us-west-1"},
			region:         "europe-east-1",
			expectedWanted: false,
		},
		{
			name: "no regions in configurations",

			confRegions:    []string{},
			region:         "us-east-1",
			expectedWanted: true,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expectedWanted, wantRegion(tt.region, tt.confRegions))
		})
	}
}
func TestWantZone(t *testing.T) {

	for _, tt := range []struct {
		name string

		confRegions    []string
		zone           string
		expectedWanted bool
	}{
		{
			name: "wanted",

			confRegions:    []string{"us-east1", "us-west1"},
			zone:           "zone/us-east1-c",
			expectedWanted: true,
		},
		{
			name: "not wanted",

			confRegions:    []string{"us-east1", "us-west1"},
			zone:           "zone/europe-east1-b",
			expectedWanted: false,
		},
		{
			name: "no regions in configurations",

			confRegions:    []string{},
			zone:           "zone/us-east1-c",
			expectedWanted: true,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expectedWanted, wantZone(tt.zone, tt.confRegions))
		})
	}
}

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
	"strings"

	"cloud.google.com/go/container/apiv1/containerpb"

	"github.com/cespare/xxhash"

	"github.com/elastic/go-freelru"
)

func getResourceNameFromURL(res string) string {
	s := strings.Split(res, "/")
	return s[len(s)-1]
}

func getRegionFromZoneURL(zone string) string {
	z := getResourceNameFromURL(zone)
	r := strings.Split(z, "-")
	return strings.Join(r[:len(r)-1], "-")
}

func getVpcIdFromLink(selfLink string, vpcAssetCache *freelru.LRU[string, *vpc]) string {
	v, ok := vpcAssetCache.Get(selfLink)
	if ok {
		return v.ID
	}
	return ""
}

func getNetSelfLinkFromNetConfig(networkConfig *containerpb.NetworkConfig) string {
	network := networkConfig.Network
	if len(network) > 0 {
		return "https://www.googleapis.com/compute/v1/" + network
	}

	return ""
}

func hashStringXXHASH(s string) uint32 {
	return uint32(xxhash.Sum64String(s))
}

// region is in the form of regions/us-west2
func wantRegion(region string, confRegions []string) bool {
	if len(confRegions) == 0 {
		return true
	}
	ss := strings.Split(region, "/")
	subnetsRegion := ss[len(ss)-1]
	for _, region := range confRegions {
		if region == subnetsRegion {
			return true
		}
	}

	return false
}

func wantZone(zone string, confRegions []string) bool {
	if len(confRegions) == 0 {
		return true
	}

	region := getRegionFromZoneURL(zone)
	for _, z := range confRegions {
		if z == region {
			return true
		}
	}

	return false
}

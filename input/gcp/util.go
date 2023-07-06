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
	"time"

	"cloud.google.com/go/compute/apiv1/computepb"
	"cloud.google.com/go/container/apiv1/containerpb"
	"google.golang.org/protobuf/proto"

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

func getSubnetIdFromLink(selfLink string, subnetAssetCache *freelru.LRU[string, *subnet]) string {
	v, ok := subnetAssetCache.Get(selfLink)
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

func getTestVpcCache() *freelru.LRU[string, *vpc] {
	vpcAssetsCache, _ := freelru.New[string, *vpc](8192, hashStringXXHASH)
	nv := vpc{
		ID: "1",
	}
	selfLink := "https://www.googleapis.com/compute/v1/projects/my_project/global/networks/my_network"
	vpcAssetsCache.AddWithExpire(selfLink, &nv, 60*time.Second)
	return vpcAssetsCache
}

func getTestSubnetCache() *freelru.LRU[string, *subnet] {
	subnetAssetsCache, _ := freelru.New[string, *subnet](8192, hashStringXXHASH)
	sb := subnet{
		ID: "2",
	}
	selfLink := "https://www.googleapis.com/compute/v1/projects/elastic-observability/regions/us-central1/subnetworks/my_subnet"
	subnetAssetsCache.AddWithExpire(selfLink, &sb, 60*time.Second)
	return subnetAssetsCache
}

func getTestComputeCache() *freelru.LRU[string, *computeInstance] {
	computeAssetsCache, _ := freelru.New[string, *computeInstance](8192, hashStringXXHASH)
	cI := computeInstance{
		ID:     "123",
		Region: "europe-west1",
		RawMd: &computepb.Metadata{
			Items: []*computepb.Items{
				{
					Key:   proto.String("kube-labels"),
					Value: proto.String("cloud.google.com/gke-nodepool=mynodepool"),
				},
			},
		},
	}
	selfLink := "https://www.googleapis.com/compute/v1/projects/elastic-observability/zones/europe-west1-d/instances/my-instance-1"
	computeAssetsCache.AddWithExpire(selfLink, &cI, 60*time.Second)
	return computeAssetsCache
}

func getComputeCache() *freelru.LRU[string, *computeInstance] {
	computeAssetsCache, _ := freelru.New[string, *computeInstance](8192, hashStringXXHASH)
	return computeAssetsCache
}

func getSubnetCache() *freelru.LRU[string, *subnet] {
	computeAssetsCache, _ := freelru.New[string, *subnet](8192, hashStringXXHASH)
	return computeAssetsCache
}

func getVpcCache() *freelru.LRU[string, *vpc] {
	computeAssetsCache, _ := freelru.New[string, *vpc](8192, hashStringXXHASH)
	return computeAssetsCache
}

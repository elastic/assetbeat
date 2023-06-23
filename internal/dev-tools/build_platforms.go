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

package dev_tools

import (
	"fmt"
	"golang.org/x/exp/slices"
	"os"
	"strings"
)

type Platform struct {
	GOOS   string
	GOARCH string
}

var supportedPlatforms = []string{"linux/amd64", "linux/arm64"}

// GetPlatforms return the list of Platform to use for cross-builds.
// By default, it returns the list of supported platforms. It can be overridden by setting the PLATFORMS
// environment variable.
func GetPlatforms() []Platform {
	var platformsList []Platform
	platforms, ok := os.LookupEnv("PLATFORMS")
	if ok {
		platformsList = getPlatformsList(platforms)
	} else {
		fmt.Println("PLATFORMS env variable not defined.")
		for _, platform := range supportedPlatforms {
			platformsList = append(platformsList, newPlatform(platform))
		}
	}
	fmt.Printf("Platforms: %+v\n", platformsList)
	return platformsList
}

// getPlatformsList returns a list of Platform from a space-delimited string of GOOS/GOARCH pairs.
// If the Platform is not supported, it is discarded from the returned list
func getPlatformsList(platforms string) []Platform {
	var platformsList []Platform
	inputPlatformsList := strings.Split(platforms, " ")
	for _, platform := range inputPlatformsList {
		if slices.Contains(supportedPlatforms, platform) {
			platformsList = append(platformsList, newPlatform(platform))
		} else {
			fmt.Printf("Unsupported platform %s. Skipping...", platform)
		}
	}
	return platformsList
}

// newPlatform returns a new Platform from a GOOS/GOARCH string
func newPlatform(p string) Platform {
	platformSplit := strings.Split(p, "/")
	return Platform{
		GOOS:   platformSplit[0],
		GOARCH: platformSplit[1],
	}
}

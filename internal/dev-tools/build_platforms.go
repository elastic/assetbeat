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

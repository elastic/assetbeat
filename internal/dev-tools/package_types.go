package dev_tools

import (
	"fmt"
	"golang.org/x/exp/slices"
	"os"
	"strings"
)

var supportedPackageTypes = []string{"docker", "tar.gz"}

// GetPackageTypes return the list of package types to use for packaging/release distribution.
// By default, it returns the list of supported package types. It can be overridden by setting the TYPES
// environment variable.
func GetPackageTypes() []string {
	var packageTypesList []string
	types, ok := os.LookupEnv("TYPES")
	if ok {
		packageTypesList = getPackageTypesList(types)
	} else {
		fmt.Println("TYPES env variable not defined.")
		packageTypesList = append(packageTypesList, supportedPackageTypes...)
	}
	fmt.Printf("PackageTypes: %+v\n", packageTypesList)
	return packageTypesList
}

// getPackageTypesList returns a list of package types from a space-delimited string of package types
// If the package type is not supported, it is discarded from the returned list
func getPackageTypesList(types string) []string {
	var typesList []string
	inputTypesList := strings.Split(types, " ")
	for _, packageType := range inputTypesList {
		if slices.Contains(supportedPackageTypes, packageType) {
			typesList = append(typesList, packageType)
		} else {
			fmt.Printf("Unsupported packageType %s. Skipping...", packageType)
		}
	}
	return typesList
}

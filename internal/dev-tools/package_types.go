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

var supportedPackageTypes = []string{"docker", "tar.gz"}

// GetPackageTypes returns the list of package types to use for packaging/release distribution.
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

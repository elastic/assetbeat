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
	"github.com/elastic/assetbeat/version"
	"github.com/magefile/mage/sh"
	"path/filepath"
	"strings"
)

const assetbeatModulePath = "github.com/elastic/assetbeat"

var qualifierVarPath = assetbeatModulePath + "/version.buildQualifier"
var defaultCrossBuildFolder = filepath.Join("build", "golang-crossbuild")

type BuildArgs struct {
	name         string //name of the binary
	targetFolder string
	flags        []string
	env          map[string]string
	ldflags      []string
}

// DefaultBuildArgs returns the default BuildArgs for use in builds.
func DefaultBuildArgs() BuildArgs {

	args := BuildArgs{
		name:         "assetbeat",
		targetFolder: "",
		// -trimpath -> remove all file system paths from the resulting executable.
		// E.g a stack trace for /home/me/stuff/src/github.com/me/something.go:9 would be shown as github.com/me/something.go:9
		flags: []string{"-trimpath"},
		// -ldflags=-s -w -> removes debug symbols from the resulting executable, reducing its size.
		ldflags: []string{"-s", "-w"},
		env:     map[string]string{},
	}

	if version.HasQualifier {
		args.ldflags = append(args.ldflags, fmt.Sprintf("-X %s=%s", qualifierVarPath, version.Qualifier))
	}

	return args
}

// DefaultCrossBuildArgs returns the default BuildArgs for cross-builds of a specific Platform.
func DefaultCrossBuildArgs(platform Platform) BuildArgs {
	args := DefaultBuildArgs()
	args.targetFolder = defaultCrossBuildFolder
	args.name = strings.Join([]string{"assetbeat", platform.GOOS, platform.GOARCH}, "-")

	args.env = map[string]string{
		"GOOS":        platform.GOOS,
		"GOARCH":      platform.GOARCH,
		"CGO_ENABLED": "0",
	}
	return args
}

// Build builds assetbeat using the defined BuildArgs and returns the executable file path.
func Build(args BuildArgs) (string, error) {

	if err := sh.RunV("go", "mod", "download"); err != nil {
		return "", err
	}

	if len(args.targetFolder) > 0 {
		if err := sh.RunV("mkdir", "-p", args.targetFolder); err != nil {
			return "", err
		}
	}

	executablePath := filepath.Join(args.targetFolder, args.name)
	buildArgs := []string{"build"}
	buildArgs = append(buildArgs, "-o", executablePath)
	buildArgs = append(buildArgs, args.flags...)
	ldflags := strings.Join(args.ldflags, " ")
	buildArgs = append(buildArgs, "-ldflags", ldflags)
	fmt.Printf("%v+\n", buildArgs)
	err := sh.RunWithV(args.env, "go", buildArgs...)
	if err != nil {
		return "", nil
	}
	return executablePath, nil
}

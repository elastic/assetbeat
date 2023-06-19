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

//go:build mage

package main

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

// Format formats all source files with `go fmt`
func Format() error {
	if err := sh.RunV("go", "fmt", "./..."); err != nil {
		return err
	}

	if os.Getenv("CI") == "true" {
		// fails if there are changes
		if err := sh.RunV("git", "diff", "--quiet"); err != nil {
			return fmt.Errorf("There are unformatted files; run `mage format` locally and commit the changes to fix.")
		}
	}

	return nil
}

// Build downloads dependencies and builds the assetbeat binary
func Build() error {
	if err := sh.RunV("go", "mod", "download"); err != nil {
		return err
	}

	return sh.RunV("go", "build", ".")
}

// Lint runs golangci-lint
func Lint() error {
	err := installTools()
	if err != nil {
		return err
	}

	fmt.Println("Running golangci-lint...")
	return sh.RunV("./.tools/golangci-lint", "run")
}

func AddLicenseHeaders() error {
	err := installTools()
	if err != nil {
		return err
	}
	fmt.Println("adding license headers with go-licenser...")
	return sh.RunV("./.tools/go-licenser", "-license", "ASL2")
}

func CheckLicenseHeaders() error {
	err := installTools()
	if err != nil {
		return err
	}
	fmt.Println("checking license headers with go-licenser...")
	return sh.RunV("./.tools/go-licenser", "-d", "-license", "ASL2")
}

// UnitTest runs all unit tests and writes a HTML coverage report to the build directory
func UnitTest() error {
	coverageFile := "coverage-unit-tests.out"
	coverageThreshold := 45

	fmt.Println("Running unit tests...")
	if err := sh.RunV("go", "test", "./...", "-coverprofile="+coverageFile); err != nil {
		return err
	}

	fmt.Println("Generating HTML coverage report...")
	if err := generateHTMLCoverageReport(coverageFile, "coverage-unit-tests.html"); err != nil {
		// not a fatal error
		fmt.Fprintf(os.Stderr, "could not generate HTML coverage report\n")
	}

	fmt.Println("Checking coverage threshold...")
	aboveThreshold, err := isCoveragePercentageIsAboveThreshold(coverageFile, coverageThreshold)
	if err != nil {
		// we need to be able to check the coverage for the build to succeed
		return fmt.Errorf("could not check coverage against threshold: %w", err)
	}

	if !aboveThreshold {
		return fmt.Errorf("code coverage did not meet required threshold of %d%%", coverageThreshold)
	}

	return nil
}

// IntegrationTest runs all integration tests
func E2ETest() error {
	fmt.Println("Running end-to-end tests...")
	return sh.RunV("go", "test", "github.com/elastic/assetbeat/tests/e2e", "-tags=e2e")
}

func generateHTMLCoverageReport(coverageFile, htmlFile string) error {
	return sh.RunV("go", "tool", "cover", "-html="+coverageFile, "-o", htmlFile)
}

func isCoveragePercentageIsAboveThreshold(coverageFile string, thresholdPercent int) (bool, error) {
	report, err := sh.Output("go", "tool", "cover", "-func="+coverageFile)
	if err != nil {
		return false, err
	}

	reportLines := strings.Split(report, "\n")
	coverageSummary := strings.Fields(reportLines[len(reportLines)-1])
	if len(coverageSummary) != 3 || !strings.HasSuffix(coverageSummary[2], "%") {
		return false, fmt.Errorf("could not parse coverage report; summary line in unexpected format")
	}

	coverage, err := strconv.ParseInt(coverageSummary[2][:2], 10, 8)
	if err != nil {
		return false, fmt.Errorf("could not parse coverage report; summary percentage could not be converted to int")
	}

	return int(coverage) >= thresholdPercent, nil
}

func installTools() error {
	fmt.Println("Installing tools...")
	oldPath, _ := os.Getwd()
	toolsPath := oldPath + "/internal/tools"
	os.Chdir(toolsPath)
	defer os.Chdir(oldPath)

	if err := sh.RunV("go", "mod", "download"); err != nil {
		return err
	}

	tools, err := sh.Output("go", "list", "-f", "{{range .Imports}}{{.}} {{end}}", "tools.go")
	if err != nil {
		return err
	}

	return sh.RunWithV(map[string]string{"GOBIN": oldPath + "/.tools"}, "go", append([]string{"install"}, strings.Fields(tools)...)...)
}

// Package packages assetbeat for distribution
// Use PLATFORMS to control the target platforms. Only linux/amd64 is supported.
// Use TYPES to control the target Type. Only Docker is supported
// Example of Usage: PLATFORMS=linux/amd64 TYPES=docker mage package
func Package() error {
	start := time.Now()
	defer func() { fmt.Println("package ran for", time.Since(start)) }()

	platform, ok := os.LookupEnv("PLATFORMS")
	if !ok {
		return fmt.Errorf("PLATFORMS env var is not set. Available options are %s", "linux/amd64")
	}
	types, ok := os.LookupEnv("TYPES")
	if !ok {
		return fmt.Errorf("TYPES env var is not set. Available options are %s", "docker")
	}

	fmt.Printf("package command called for Platforms=%s and TYPES=%s\n", platform, types)
	if platform == "linux/amd64" && types == "docker" {
		filePath := "build/package/assetbeat/assetbeat-linux-amd64.docker/docker-build"
		executable := filePath + "/assetbeat"
		dockerfile := filePath + "/Dockerfile"

		fmt.Printf("Creating filepath %s\n", filePath)
		if err := sh.RunV("mkdir", "-p", filePath); err != nil {
			return err
		}
		var envMap = map[string]string{
			"GOOS":   "linux",
			"GOARCH": "amd64",
		}
		fmt.Println("Building assetbeat binary")
		if err := sh.RunWithV(envMap, "go", "build", "-o", executable); err != nil {
			return err
		}
		fmt.Println("Copying Dockerfile")
		if err := sh.RunV("cp", "Dockerfile.reference", dockerfile); err != nil {
			return err
		}
	}

	return nil
}

// GetVersion returns the version of assetbeat
// in the format of 'assetbeat version 8.7.0 (amd64), libbeat 8.7.0 [unknown built unknown]'
func GetVersion() error {
	_, version, err := getVersion()
	if err != nil {
		return err
	}

	fmt.Println(version)
	return nil
}

// WriteVersionToGithubOutput appends the assetbeat version to $GITHUB_OUTPUT
// environment file in the format of VERSION=8.7.0
// Its purpose is to be used by Github Actions
// https://docs.github.com/en/actions/using-jobs/defining-outputs-for-jobs
func WriteVersionToGithubOutput() error {
	shortVersion, _, err := getVersion()
	if err != nil {
		return err
	}
	return writeOutput(fmt.Sprintf("VERSION=%s\n", shortVersion))
}

// getVersion returns the assetbeat long and short version
// example: shortVersion:8.7.0,
// longVersion: assetbeat version 8.7.0 (amd64), libbeat 8.7.0 [unknown built unknown]
func getVersion() (shortVersion string, longVersion string, err error) {
	mg.Deps(Build)

	longVersion, err = sh.Output("./assetbeat", "version")
	if err != nil {
		return
	}

	awk := exec.Command("awk", "$2 = \"version\" {printf $3}")
	awk.Stdin = strings.NewReader(longVersion)

	out, err := awk.Output()
	if err != nil {
		return
	}

	shortVersion = string(out)
	return
}

// writeOutput writes a key,value string to Github's
// output env file $GITHUB_OUTPUT
func writeOutput(output string) error {
	file, exists := os.LookupEnv("GITHUB_OUTPUT")

	if exists {
		fw, err := os.OpenFile(file, os.O_APPEND|os.O_WRONLY, 0600)
		if err != nil {
			return err
		}
		defer fw.Close()
		if _, err := fw.WriteString(output); err != nil {
			return err
		}
	}

	return nil
}

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
	"github.com/elastic/elastic-agent-libs/dev-tools/mage"
	"github.com/elastic/elastic-agent-libs/dev-tools/mage/gotool"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	devtools "github.com/elastic/assetbeat/internal/dev-tools"
	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

// Format formats all source files with `go fmt`
func Format() error {
	return sh.RunV("go", "fmt", "./...")
}

// Clean removes the build directory.
func Clean() {
	os.RemoveAll("build") // nolint:errcheck //not required
}

// Build builds the assetbeat binary with the default build arguments
func Build() error {
	_, error := devtools.Build(devtools.DefaultBuildArgs())
	return error
}

// Lint runs golangci-lint
func Lint() error {
	return mage.Linter{}.All()
}

// Notice generates a NOTICE.txt file for the module.
func Notice() error {
	return devtools.GenerateNotice(
		filepath.Join("internal", "notice", "overrides.json"),
		filepath.Join("internal", "notice", "rules.json"),
		filepath.Join("internal", "notice", "NOTICE.txt.tmpl"),
	)
}

// DependencyReport generates a dependencies.csv file for the module.
func DependencyReport() error {
	return devtools.GenerateDependencyReport(
		filepath.Join("internal", "notice", "overrides.json"),
		filepath.Join("internal", "notice", "rules.json"),
		filepath.Join("internal", "notice", "dependencies.csv.tmpl"),
		isSnapshot(),
	)
}

// Check runs all the checks including licence, notice, gomod, git changes
func Check() {
	// these are not allowed in parallel
	mg.SerialDeps(
		Lint,
		Format,
		CheckLicenseHeaders,
		Notice,
		mage.Deps.CheckModuleTidy,
		mage.CheckNoChanges,
	)
}

// Update updates go.mod, and runs format, addLicenseHeaders and notice
func Update() {
	mg.SerialDeps(gotool.Mod.Tidy, Format, AddLicenseHeaders, Notice)
}

// AddLicenseHeaders add a license header to any *.go file where it is missing
func AddLicenseHeaders() error {
	mg.Deps(mage.InstallGoLicenser)
	return gotool.Licenser(
		gotool.Licenser.License("ASL2"),
	)
}

// CheckLicenseHeaders check if all the *.go files have a license header
func CheckLicenseHeaders() error {
	mg.Deps(mage.InstallGoLicenser)

	return gotool.Licenser(
		gotool.Licenser.License("ASL2"),
		gotool.Licenser.Check(),
	)
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

// E2ETest runs all end-to-end tests
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

// Package packages assetbeat for distribution
// Use PLATFORMS to control the target platforms. Only linux/amd64 and linux/arm64 are supported.
// Use TYPES to control the target Type. Only tar.gz and Docker are supported.
// Example of Usage: PLATFORMS=linux/amd64 TYPES=docker mage package
func Package() error {
	start := time.Now()
	defer func() { fmt.Println("package ran for", time.Since(start)) }()

	for _, platform := range devtools.GetPlatforms() {
		executablePath, err := devtools.Build(devtools.DefaultCrossBuildArgs(platform))
		if err != nil {
			return err
		}
		for _, packageType := range devtools.GetPackageTypes() {
			fmt.Printf(">>>>> Packaging assetbeat for platform: %+v packageType:%s\n", platform, packageType)
			packageSpec := devtools.PackageSpec{
				Os:             platform.GOOS,
				Arch:           platform.GOARCH,
				PackageType:    packageType,
				ExecutablePath: executablePath,
				IsSnapshot:     isSnapshot(),
				ExtraFilesList: devtools.GetDefaultExtraFiles(),
			}
			err = devtools.CreatePackage(packageSpec)
			if err != nil {
				return err
			}
		}
	}
	return DependencyReport()
}

func isSnapshot() bool {
	isSnapshot, ok := os.LookupEnv("SNAPSHOT")
	if ok {
		return isSnapshot == "true"
	}
	return false
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

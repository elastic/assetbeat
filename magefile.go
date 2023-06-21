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
	"archive/tar"
	"compress/gzip"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"io"
	"k8s.io/utils/strings/slices"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	settings "github.com/elastic/assetbeat/cmd"
	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

type Platform struct {
	GOOS   string
	GOARCH string
}

var supportedPlatforms = []string{"linux/amd64", "linux/arm64"}
var supportedPackageTypes = []string{"docker", "tar.gz"}
var defaultCrossBuildFolder = filepath.Join("build", "golang-crossbuild")
var defaultPackageFolder = filepath.Join("build", "distributions")

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

	return sh.RunV("go", "build", "-ldflags=-s -w", "-trimpath", ".")
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

	var platformsList []string
	var typesList []string

	platforms, ok := os.LookupEnv("PLATFORMS")
	fmt.Printf("package command called for Platforms=%s\n", platforms)
	if ok {
		platformsList = getPlatformsList(platforms)
	} else {
		fmt.Printf("PLATFORMS env variable not defined. Package will run for all supported platforms %+v", supportedPlatforms)
		platformsList = supportedPlatforms
	}

	types, ok := os.LookupEnv("TYPES")
	fmt.Printf("package command called for Package Types=%s\n", types)
	if ok {
		typesList = getTypesList(types)
	} else {
		fmt.Printf("TYPES env variable not defined. Package will run for all supported package types %+v", supportedPackageTypes)
		typesList = supportedPackageTypes
	}

	for _, fullPlatform := range platformsList {
		platform := getPlatform(fullPlatform)
		for _, packageType := range typesList {
			fmt.Printf("Packaging assetbeat for platform: %s packageType:%s\n", fullPlatform, packageType)
			executablePath, err := crossBuild(platform)
			if err != nil {
				return err
			}
			if packageType == "docker" {
				filePath := fmt.Sprintf("build/package/assetbeat/assetbeat-%s-%s.docker/docker-build", platform.GOOS, platform.GOARCH)
				dockerfile := filePath + "/Dockerfile"
				executable := filePath + "/assetbeat"

				fmt.Printf("Creating filepath %s\n", filePath)
				if err := sh.RunV("mkdir", "-p", filePath); err != nil {
					return err
				}

				fmt.Println("Copying Executable")
				if err := sh.RunV("cp", executablePath, executable); err != nil {
					return err
				}

				fmt.Println("Copying Dockerfile")
				if err := sh.RunV("cp", "Dockerfile.reference", dockerfile); err != nil {
					return err
				}
			} else {
				filesPathList := []string{executablePath, "LICENSE.txt", "README.md", "assetbeat.yml"}
				tarFileNameElements := []string{"assetbeat", settings.Version}
				if isSnapshot() {
					tarFileNameElements = append(tarFileNameElements, "SNAPSHOT")
				}
				tarFileNameElements = append(tarFileNameElements, []string{platform.GOOS, platform.GOARCH}...)
				tarFileName := strings.Join(tarFileNameElements, "-") + ".tar.gz"
				if err := sh.RunV("mkdir", "-p", defaultPackageFolder); err != nil {
					return err
				}
				tarFilePath := filepath.Join(defaultPackageFolder, tarFileName)
				err := createTarball(tarFilePath, filesPathList)
				if err != nil {
					return err
				}
				err = createSHA512File(tarFilePath)
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// CreateSHA512File computes the sha512 sum of the specified file the writes
// a sidecar file containing the hash and filename.
func createSHA512File(file string) error {
	fmt.Printf("Creating SHA512 hash... Filepath: %s\n", file+".sha512")
	f, err := os.Open(file)
	if err != nil {
		return fmt.Errorf("failed to open file for sha512 summing. Error %s", err)
	}
	defer f.Close()

	sum := sha512.New()
	if _, err := io.Copy(sum, f); err != nil {
		return fmt.Errorf("failed reading from input file. Error %s", err)
	}

	computedHash := hex.EncodeToString(sum.Sum(nil))
	out := fmt.Sprintf("%v  %v", computedHash, filepath.Base(file))

	return os.WriteFile(file+".sha512", []byte(out), 0644)
}

func isSnapshot() bool {
	isSnapshot, ok := os.LookupEnv("SNAPSHOT")
	if ok {
		return isSnapshot == "true"
	}
	return false
}

func createTarball(tarballFilePath string, filePaths []string) error {
	fmt.Printf("Creating tarball... Filepath: %s\n", tarballFilePath)
	file, err := os.Create(tarballFilePath)
	if err != nil {
		return fmt.Errorf("could not create tarball file '%s', got error '%s'", tarballFilePath, err)
	}
	defer file.Close()

	gzipWriter := gzip.NewWriter(file)
	defer gzipWriter.Close()

	tarWriter := tar.NewWriter(gzipWriter)
	defer tarWriter.Close()

	for _, filePath := range filePaths {
		err := addFileToTarWriter(filePath, tarWriter)
		if err != nil {
			return fmt.Errorf("could not add file '%s', to tarball, got error '%s'", filePath, err)
		}
	}

	return nil
}

func addFileToTarWriter(filePath string, tarWriter *tar.Writer) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("could not open file '%s', got error '%s'", filePath, err)
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return fmt.Errorf("could not get stat for file '%s', got error '%s'", filePath, err)
	}

	headerName := filepath.Base(filePath)
	if strings.Contains(headerName, "assetbeat") {
		//This makes sure that platform details are removed from the packaged assetbeat binary filename
		headerName = "assetbeat"
	}
	header := &tar.Header{
		Name:    headerName,
		Size:    stat.Size(),
		Mode:    int64(stat.Mode()),
		ModTime: stat.ModTime(),
	}

	err = tarWriter.WriteHeader(header)
	if err != nil {
		return fmt.Errorf("could not write header for file '%s', got error '%s'", filePath, err)
	}

	_, err = io.Copy(tarWriter, file)
	if err != nil {
		return fmt.Errorf("could not copy the file '%s' data to the tarball, got error '%s'", filePath, err)
	}

	return nil
}

func crossBuild(platform Platform) (string, error) {
	fmt.Printf("Building assetbeat binary for platform %+v\n", platform)
	if err := sh.RunV("mkdir", "-p", defaultCrossBuildFolder); err != nil {
		return "", err
	}
	executable := strings.Join([]string{"assetbeat", platform.GOOS, platform.GOARCH}, "-")

	envMap := map[string]string{
		"GOOS":   platform.GOOS,
		"GOARCH": platform.GOARCH,
	}
	executablePath := filepath.Join(defaultCrossBuildFolder, executable)
	err := sh.RunWithV(envMap, "go", "build", "-o", executablePath, "-trimpath", "-ldflags=-s -w")
	if err != nil {
		return "", nil
	}
	return executablePath, nil
}

func getPlatformsList(platforms string) []string {
	var platformsList []string
	inputPlatformsList := strings.Split(platforms, " ")
	for _, platform := range inputPlatformsList {
		if slices.Contains(supportedPlatforms, platform) {
			platformsList = append(platformsList, platform)
		} else {
			fmt.Printf("Unsupported platform %s. Skipping...", platform)
		}
	}
	return platformsList
}

func getTypesList(types string) []string {
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

func getPlatform(p string) Platform {
	platformSplit := strings.Split(p, "/")
	return Platform{
		GOOS:   platformSplit[0],
		GOARCH: platformSplit[1],
	}
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

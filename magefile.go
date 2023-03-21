//go:build mage

package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

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

// Build downloads dependencies and builds the inputrunner binary
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
	return sh.RunV("go", "test", "github.com/elastic/inputrunner/tests/e2e", "-tags=e2e")
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

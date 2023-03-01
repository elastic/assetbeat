//go:build mage

package main

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

// Format formats all source files with `go fmt`
func Format() error {
	if err := sh.RunV("go", "fmt", "./..."); err != nil {
		return err
	}

	// fails if there are changes
	if err := sh.RunV("git", "diff", "--quiet"); err != nil {
		return fmt.Errorf("There are unformatted files; run `mage format` and commit your changes to fix.")
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

// Check runs static analysis and security checks
func Check() error {
	mg.Deps(staticcheck, gosec)
	return nil
}

// UnitTest runs all unit tests and writes a HTML coverage report to the build directory
func UnitTest() error {
	coverageFile := "coverage-unit-tests.out"
	coverageThreshold := 40

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
func IntegrationTest() error {
	fmt.Println("Running integration tests...")
	return sh.RunV("go", "test", "./...", "-tags=integration")
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

func staticcheck() error {
	if installed := install("staticcheck", "honnef.co/go/tools/cmd/staticcheck@latest"); !installed {
		return nil
	}

	fmt.Println("Running staticcheck...")
	return sh.RunV("staticcheck", "-f=stylish", "./...")
}

func gosec() error {
	if installed := install("gosec", "github.com/securego/gosec/v2/cmd/gosec@latest"); !installed {
		return nil
	}

	fmt.Println("Running gosec...")
	return sh.RunV("gosec", "./...")
}

func install(packageName, installURL string) (isInstalled bool) {
	_, missing := exec.LookPath(packageName)
	if missing != nil {
		fmt.Printf("installing %v...\n", packageName)
		err := sh.RunV("go", "install", installURL)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Could not install %v, skipping...\n", packageName)
			return false
		}
		fmt.Printf("%v installed...\n", packageName)
	}
	return true
}

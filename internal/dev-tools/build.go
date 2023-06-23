package dev_tools

import (
	"github.com/magefile/mage/sh"
	"path/filepath"
	"strings"
)

var defaultCrossBuildFolder = filepath.Join("build", "golang-crossbuild")

type BuildArgs struct {
	name         string //name of the binary
	targetFolder string
	flags        []string
	env          map[string]string
}

// DefaultBuildArgs returns the default BuildArgs for use in builds.
func DefaultBuildArgs() BuildArgs {
	return BuildArgs{
		name:         "assetbeat",
		targetFolder: "",
		// -trimpath -> remove all file system paths from the resulting executable.
		// E.g a stack trace for /home/me/stuff/src/github.com/me/something.go:9 would be shown as github.com/me/something.go:9
		//
		// -ldflags=-s -w -> removes debug symbols from the resulting executable, reducing its size.
		flags: []string{"-trimpath", "-ldflags=-s -w"},
		env:   map[string]string{},
	}
}

// DefaultCrossBuildArgs returns the default BuildArgs for cross-builds of a specific Platform.
func DefaultCrossBuildArgs(platform Platform) BuildArgs {
	args := DefaultBuildArgs()
	args.targetFolder = defaultCrossBuildFolder
	args.name = strings.Join([]string{"assetbeat", platform.GOOS, platform.GOARCH}, "-")

	args.env = map[string]string{
		"GOOS":   platform.GOOS,
		"GOARCH": platform.GOARCH,
	}
	return args
}

// Build builds assetbeat using the defined BuildArgs
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
	err := sh.RunWithV(args.env, "go", buildArgs...)
	if err != nil {
		return "", nil
	}
	return executablePath, nil
}

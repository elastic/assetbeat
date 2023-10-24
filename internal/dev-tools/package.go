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

var defaultPackageFolder = filepath.Join("build", "distributions")

type PackageSpec struct {
	Os             string
	Arch           string
	PackageType    string
	ExecutablePath string
	IsSnapshot     bool
	ExtraFilesList []string
}

var packageArchOverrides = map[string]string{
	"amd64": "x86_64",
}

func GetPackageArch(goarch string) string {
	arch, overrideExists := packageArchOverrides[goarch]
	if overrideExists {
		return arch
	}
	return goarch
}

// GetDefaultExtraFiles returns the default list of files to include in an assetbeat package,
// in addition to assetbeat's executable
func GetDefaultExtraFiles() []string {
	return []string{"assetbeat.yml", "assetbeat.spec.yml"}
}

// CreatePackage assetbeat for distribution. It generates packages based on the provided PackageSpec/
func CreatePackage(spec PackageSpec) error {
	switch spec.PackageType {
	case "docker":
		return packageDocker(spec)
	case "tar.gz":
		return packageTar(spec)
	default:
		return fmt.Errorf("unsupported package type %s", spec.PackageType)
	}
}

func packageDocker(spec PackageSpec) error {
	filePath := fmt.Sprintf("build/package/assetbeat/assetbeat-%s-%s.docker/docker-build", spec.Os, spec.Arch)
	dockerfile := filePath + "/Dockerfile"
	executable := filePath + "/assetbeat"

	fmt.Printf("Creating folder %s\n", filePath)
	if err := sh.RunV("mkdir", "-p", filePath); err != nil {
		return err
	}

	fmt.Println("Copying Executable")
	if err := sh.RunV("cp", spec.ExecutablePath, executable); err != nil {
		return err
	}

	fmt.Println("Copying Dockerfile")
	return sh.RunV("cp", "Dockerfile.reference", dockerfile)
}

func packageTar(spec PackageSpec) error {
	filesPathList := []string{spec.ExecutablePath}
	filesPathList = append(filesPathList, spec.ExtraFilesList...)

	if err := sh.RunV("mkdir", "-p", defaultPackageFolder); err != nil {
		return err
	}

	basePackageName := getPackageBaseName(spec)
	tarFilePath := filepath.Join(defaultPackageFolder, basePackageName+".tar.gz")
	err := CreateTarball(basePackageName, tarFilePath, filesPathList)
	if err != nil {
		return err
	}
	return CreateSHA512File(tarFilePath)
}

func getPackageBaseName(spec PackageSpec) string {
	tarFileNameElements := []string{"assetbeat", version.GetVersion()}
	if spec.IsSnapshot {
		tarFileNameElements = append(tarFileNameElements, "SNAPSHOT")
	}
	tarFileNameElements = append(tarFileNameElements, []string{spec.Os, spec.Arch}...)

	return strings.Join(tarFileNameElements, "-")
}

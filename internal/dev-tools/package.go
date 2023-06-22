package dev_tools

import (
	"fmt"
	settings "github.com/elastic/assetbeat/cmd"
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

// GetDefaultExtraFiles returns the default list of files to include in an assetbeat package,
// in addition to assetbeat's executable
func GetDefaultExtraFiles() []string {
	return []string{"LICENSE.txt", "README.md", "assetbeat.yml"}
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

	tarFileName := getPackageTarName(spec)
	tarFilePath := filepath.Join(defaultPackageFolder, tarFileName)
	err := CreateTarball(tarFilePath, filesPathList)
	if err != nil {
		return err
	}
	return CreateSHA512File(tarFilePath)
}

func getPackageTarName(spec PackageSpec) string {
	tarFileNameElements := []string{"assetbeat", settings.Version}
	if spec.IsSnapshot {
		tarFileNameElements = append(tarFileNameElements, "SNAPSHOT")
	}
	tarFileNameElements = append(tarFileNameElements, []string{spec.Os, spec.Arch}...)

	return strings.Join(tarFileNameElements, "-") + ".tar.gz"
}

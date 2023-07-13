package version

import "os"

// This variable needs to be overriden at build time, if a version qualifier is provided
var buildQualifier = ""

var Qualifier, HasQualifier = os.LookupEnv("VERSION_QUALIFIER")

func GetBuildVersion() string {
	if buildQualifier == "" {
		return defaultVersion
	}
	return defaultVersion + "-" + buildQualifier
}

func GetVersion() string {
	if Qualifier == "" {
		return defaultVersion
	}
	return defaultVersion + "-" + Qualifier
}

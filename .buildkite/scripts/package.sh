#!/bin/bash
set -uxo pipefail

export PLATFORMS="linux/amd64 linux/arm64"
export TYPES="tar.gz"

WORKFLOW=$1

if [ "$WORKFLOW" = "snapshot" ] ; then
    export SNAPSHOT="true"
fi


# Install prerequirements (go, mage...)
source .buildkite/scripts/install-prereq.sh

# Download Go dependencies
go mod download

# Packaging the assetbeat binary
mage package

# Generate the CSV dependency report
mage dependencyReport
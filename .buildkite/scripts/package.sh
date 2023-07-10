#!/usr/bin/env bash
set -uxo pipefail

export PLATFORMS="linux/amd64 linux/arm64"
export TYPES="tar.gz"

WORKFLOW=$1

if [ "$WORKFLOW" = "snapshot" ] ; then
    export SNAPSHOT="true"
fi

# Install prerequirements (go, mage...)
MY_DIR=$(dirname $(readlink -f "$0"))
source $MY_DIR/install-prereq.sh

# Download Go dependencies
go mod download

# Packaging the assetbeat binary
# mage package

echo $PWD
echo $BUILDKITE_BUILD_CHECKOUT_PATH

# Generate the CSV dependency report
mage dependencyReport
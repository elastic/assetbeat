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
ls -alrt

git --git-dir="$PWD/.git" rev-parse > /dev/null 2>&1
# Generate the CSV dependency report
mage dependencyReport
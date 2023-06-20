#!/usr/bin/env bash
set -uxo pipefail

env
export PLATFORMS="linux/amd64 linux/arm64"
export TYPES="tar.gz"

WORKFLOW=$1

if [ "$WORKFLOW" = "snapshot" ] ; then
    export SNAPSHOT="true"
fi

# Install prerequirements (go, mage...)
MY_DIR=$(dirname $(readlink -f "$0"))
source $MY_DIR/install-prereq.sh

#Packaging the assetbeat binary
mage package
#!/usr/bin/env bash
set -euxo pipefail

# Install prerequirements (go, mage...)
MY_DIR=$(dirname $(readlink -f "$0"))
source $MY_DIR/install-prereq.sh

#Building the assetbeat binary
echo "Running unitTests"
mage unitTest
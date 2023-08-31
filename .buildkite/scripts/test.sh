#!/bin/bash
set -uxo pipefail

# Install prerequirements (go, mage...)
source .buildkite/scripts/install-prereq.sh

# Check
mage unitTest

mage e2eTest
#!/bin/bash
set -uxo pipefail

# Install prerequirements (go, mage...)
source .buildkite/scripts/install-prereq.sh

# Unit tests
mage unitTest

# End to end tests
mage e2eTest
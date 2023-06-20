#!/usr/bin/env bash
set -euxo pipefail

# Install mage
if ! command -v mage &>/dev/null; then
  go install github.com/magefile/mage@latest
else
  echo "Mage is already installed. Skipping installation..."
fi

# To avoid the following error while executing go build
# " error obtaining VCS status: exit status 128. Use -buildvcs=false to disable VCS stamping. "
go env -w GOFLAGS="-buildvcs=false"
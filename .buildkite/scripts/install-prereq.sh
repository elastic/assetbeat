#!/usr/bin/env bash
set -euxo pipefail

# Install Go
if ! command -v go &>/dev/null; then
  echo "Go is not installed. Installing Go..."
  # shellcheck disable=SC2006
  # shellcheck disable=SC2155
  export GO_VERSION=`cat .go-version`
  curl -O https://dl.google.com/go/go"$GO_VERSION".linux-amd64.tar.gz
  tar -xf go"$GO_VERSION".linux-amd64.tar.gz -C "$HOME"
  # shellcheck disable=SC2016
  echo 'export PATH=$PATH:$HOME/go/bin' >> ~/.bashrc
  # shellcheck disable=SC1090
  source ~/.bashrc
  echo "Go has been installed."
else
  echo "Go is already installed. Skipping installation..."
fi

# Install mage
if ! command -v mage &>/dev/null; then
  go install github.com/magefile/mage@latest
else
  echo "Mage is already installed. Skipping installation..."
fi
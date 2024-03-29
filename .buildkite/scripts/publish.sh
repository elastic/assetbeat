#!/bin/bash
set -euo pipefail

WORKFLOW=$1
# The "branch" here selects which "$BRANCH.gradle" file of release manager is used
export VERSION=$(grep defaultVersion version/version.go | cut -f2 -d "\"" | tail -n1)
MAJOR=$(echo $VERSION | awk -F. '{ print $1 }')
MINOR=$(echo $VERSION | awk -F. '{ print $2 }')
if [ -n "$(git ls-remote --heads origin $MAJOR.$MINOR)" ] ; then
    BRANCH=$MAJOR.$MINOR
elif [ -n "$(git ls-remote --heads origin $MAJOR.x)" ] ; then
    BRANCH=$MAJOR.x
else
    BRANCH=main
fi

# Download artifacts from other stages
echo "Downloading artifacts..."
buildkite-agent artifact download "build/distributions/*" "." --step package-"${WORKFLOW}"

# Fix file permissions
chmod -R a+r build/*
chmod -R a+w build

# Shared secret path containing the dra creds for project teams
echo "Retrieving DRA crededentials..."
DRA_CREDS=$(vault kv get -field=data -format=json kv/ci-shared/release/dra-role)

# Run release-manager
echo "Running release-manager container..."
IMAGE="docker.elastic.co/infra/release-manager:latest"
docker run --rm \
  --name release-manager \
  -e VAULT_ADDR=$(echo $DRA_CREDS | jq -r '.vault_addr') \
  -e VAULT_ROLE_ID=$(echo $DRA_CREDS | jq -r '.role_id') \
  -e VAULT_SECRET_ID=$(echo $DRA_CREDS | jq -r '.secret_id') \
  --mount type=bind,readonly=false,src="${PWD}",target=/artifacts \
  "$IMAGE" \
    cli collect \
      --project assetbeat \
      --branch "${BRANCH}" \
      --commit "${BUILDKITE_COMMIT}" \
      --workflow "${WORKFLOW}" \
      --version "${VERSION}" \
      --artifact-set main
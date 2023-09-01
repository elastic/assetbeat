#!/bin/bash
set -euxo pipefail

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
# Allow other users access to read the artifacts so they are readable in the
# container
chmod a+r build/distributions/*

# Allow other users write access to create checksum files
chmod a+w build/distributions

# Shared secret path containing the dra creds for project teams
echo "Retrieving DRA crededentials..."
DRA_CREDS=$(vault kv get -field=data -format=json kv/ci-shared/release/dra-role)

# TODO: Enable as soon as everything is in place to publish artifacts to DRA
# Run release-manager
#echo "Running release-manager container..."
#IMAGE="docker.elastic.co/infra/release-manager:latest"
#docker run --rm \
#  --name release-manager \
#  -e VAULT_ADDR=$(echo $DRA_CREDS | jq -r '.vault_addr') \
#  -e VAULT_ROLE_ID=$(echo $DRA_CREDS | jq -r '.role_id') \
#  -e VAULT_SECRET_ID=$(echo $DRA_CREDS | jq -r '.secret_id') \
#  --mount type=bind,readonly=false,src="${PWD}",target=/artifacts \
#  "$IMAGE" \
#    cli collect \
#      --project assetbeat \
#      --branch "${BRANCH}" \
#      --commit "${BUILDKITE_COMMIT}" \
#      --workflow "${WORKFLOW}" \
#      --version "${VERSION}" \
#      --artifact-set main
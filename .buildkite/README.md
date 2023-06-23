# Buildkite

This README provides an overview of the Buildkite pipeline used to automate the build and publish process for Assetbeat artifacts.

**_NOTE_: The pipeline is still a work in progress and in testing phase. Frequent changes are expected**

## Artifacts

The pipeline generates the following artifacts:

- **assetbeat-ASSETBEAT_VERSION-WORKFLOW-GOOS-GOARCH.tar.gz**: This tarball includes the `assetbeat` binary and other related files (e.g LICENSE, assetbeat.yml, etc.). The supported platforms for the artifacts are linux/amd64 and linux/arm64.
- **assetbeat-ASSETBEAT_VERSION-WORKFLOW-GOOS-GOARCH.tar.gz.sha512** The sha512 hash of the above tarball.

## Triggering the Pipeline

The pipeline is triggered in the following scenarios:

- **Snapshot Builds**: A snapshot build is triggered when a pull request (PR) is opened and also when it is merged into the 'main' branch.

## Pipeline Configuration

To view the pipeline and its configuration, click [here](https://buildkite.com/elastic/assetbeat).

## Test pipeline changes locally

Buildkite provides a command line tool, named `bk`, to run pipelines locally. To perform a local run, you need to

1. [Install Buildkite agent.](https://buildkite.com/docs/agent/v3/installation)
2. [Install `bk` cli](https://github.com/buildkite/cli)
3. Execute `bk local run` inside this repo.

For more information, please click [here](https://buildkite.com/changelog/44-run-pipelines-locally-with-bk-cli)

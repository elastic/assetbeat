# Buildkite

This README provides an overview of the Buildkite pipeline used to automate the build and publish process for Cloudbeat artifacts.

## Pipeline Configuration

To view the pipeline and its configuration, click [here](https://buildkite.com/elastic/assetbeat).

## Test pipeline changes locally

Buildkite provides a command line tool, named `bk`, to run pipelines locally. To perform a local run, you need to

1. [Install Buildkite agent.](https://buildkite.com/docs/agent/v3/installation)
2. [Install `bk` cli](https://github.com/buildkite/cli)
3. Execute `bk local run` inside this repo.

For more information, please click [here](https://buildkite.com/changelog/44-run-pipelines-locally-with-bk-cli)
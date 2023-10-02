[![Build status](https://badge.buildkite.com/c02badfd4c5a879748bbc27ecfc19147c296af2410391dc749.svg?branch=main)](https://buildkite.com/elastic/assetbeat)
[![Go Report Card](https://goreportcard.com/badge/github.com/elastic/assetbeat)](https://goreportcard.com/report/github.com/elastic/assetbeat)

# Assetbeat

Assetbeat is a small binary for collecting information about infrastructure "assets". Assets are defined as elements within your infrastructure, such as containers, machines, pods, clusters, etc.

**Note:** Assetbeat is currently in technical preview and may be subject to frequent changes. Elastic will apply best effort to fix any issues, but features in technical preview are not subject to the support SLA of official GA features.

## Inputs

Documentation for each input can be found in the relevant directory (e.g. input/aws).

## Development

Requirements:
- go 1.20+
- [Mage](https://magefile.org/)

Mage targets are self-explanatory and can be listed with `mage -l`.

Build the assetbeat binary with `mage build`, and run it locally with `./assetbeat`.
See `./assetbeat -h` for more detail on configuration options.

Run `mage update` before creating new PRs. This command automatically updates `go.mod`, add license headers to any new *.go files and re-generate 
NOTICE.txt. Also double-check that `mage check` returns with no errors, as the PR CI will fail otherwise.

Please aim for 100% unit test coverage on new code.
You can view the HTML coverage report by running `mage unitTest && [xdg-]open ./coverage.html`.

### Requirements for inputs

- Compatible with [Elastic Agent v2](https://github.com/elastic/elastic-agent/blob/main/docs/architecture.md)
- No [Cgo](https://pkg.go.dev/cmd/cgo) allowed
- Stateless (including publisher)
- Config must be compatible with Elastic Agent

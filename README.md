# assetbeat

assetbeat is a small binary for running stateless [Elastic Agent v2 inputs](https://github.com/elastic/elastic-agent-inputs/issues/1).

It’s still a beat, for now.
But the intention is that this is as lightweight as possible, until the day when standalone inputs can utilise the [Elastic Agent v2 shipper](https://github.com/elastic/elastic-agent-shipper).

## Inputs

Documentation for each input can be found in the releveant directory (e.g. input/aws).

## Development

Requirements:
- go 1.19+
- [Mage](https://magefile.org/)

Mage targets are self-explanitory and can be listed with `mage -l`.

Build the assetbeat binary with `mage build`, and run it locally with `./assetbeat`.
See `./assetbeat -h` for more detail on configuration options.

Run `mage update` before creating new PRs. This command automatically updates `go.mod`, add license headers to any new *.go files and re-generate 
NOTICE.txt. Also double-check that `mage check` returns with no errors, as the PR CI will fail otherwise.

Please aim for 100% unit test coverage on new code.
You can view the HTML coverage report by running `mage unitTest && [xdg-]open ./coverage.html`.

### Requirements for inputs (WIP)

- Compatible with [Elastic Agent v2 inputs](https://github.com/elastic/elastic-agent-inputs/issues/1)
- No [Cgo](https://pkg.go.dev/cmd/cgo) allowed
- Stateless (including publisher)
- Config must be compatible with Elastic Agent

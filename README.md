# Inputrunner

Inputrunner is a small binary for running stateless [Elastic Agent v2 inputs](https://github.com/elastic/elastic-agent-inputs/issues/1).

It’s still a beat, for now.
But the intention is that this is as lightweight as possible, until the day when standalone inputs can utilise the [Elastic Agent v2 shipper](https://github.com/elastic/elastic-agent-shipper).

## Development

Requirements:
- go 1.19+
- [Mage](https://magefile.org/)

Mage targets are self-explanitory and can be listed with `mage -l`.

Build the inputrunner binary with `mage build`, and run it locally with `./inputrunner`.

### Requirements for inputs (WIP)

- Compatible with [Elastic Agent v2 inputs](https://github.com/elastic/elastic-agent-inputs/issues/1)
- No Cgo allowed
- Stateless (including publisher)
- Config must be compatible with Elastic Agent

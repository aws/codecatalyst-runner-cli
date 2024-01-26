# Development

This document describes the steps a developer uses to make changes to the CodeCatalyst Runner CLI source code. It is recommended that developers first review the [architecture](./architecture.md) before contributing for the first time. Additionally, review the [CONTRIBUTING](../CONTRIBUTING.md) documentation to understand how to create issues and pull requests. All contributions must comply with the [CODE_OF_CONDUCT](../CODE_OF_CONDUCT.md).

## Prerequisites

Verify that you have Go 1.21+ installed

```bash
go version
```

If `go` is not installed, follow the [instructions](https://go.dev/doc/install) on the Go website or install with Homebrew:

```bash
brew install go
```

Clone this repository:

```bash
git clone git@github.com:aws/codecatalyst-runner-cli.git
cd codecatalyst-runner-cli
```

Install development dependencies: `make deps`

## Building

To format all source code, run: `make format`

To run all linters, run: `make lint`

To run all tests, run: `make test`

## Running

If you want to make changes to `ccr` and test locally, you can run against source with:

```bash
go run main.go <insert command flags here>
```

Install `ccr` locally by running `make -C codecatalyst-runner install`:

## Releasing

A workflow is defined in `.github/workflows/promote.yml` to increment the patch version in `VERSION` file and also tag the commit with the new version number. This workflow runs automatically for all commits to `main` branch.

Another workflow is defined in `.github/workflows/release.yml` to build a production version of the CLI. This workflow runs automatically for new tags that match a semver pattern.

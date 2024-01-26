# Overview

This repository contains a CLI to run [Amazon CodeCatalyst workflows](https://docs.aws.amazon.com/codecatalyst/latest/userguide/flows.html) locally.

![demo](docs/ccr-demo.gif)

## Installation

Clone this repository and run: `make install`

## Usage

To execute an action against the current directory, run: `ccr -f /path/to/my/workflow.yaml`

Details usage options can be found by running `ccr -h`

```sh
Tool to run codecatalyst workflows

Usage:
  ccr [command]

Available Commands:
  completion  Generate the autocompletion script for the specified shell
  help        Help about any command

Flags:
  -a, --action string                 action to run (default: *)
  -b, --bind                          bind working directory rather than create a copy
  -c, --concurrency int               number of policies to execute concurrently (default 12)
  -n, --dryrun                        dry run
  -e, --environments stringToString   map workflow environment names to AWS CLI profile names (default [])
  -x, --executor string               executor type [docker,shell] (default "docker")
  -h, --help                          help for ccr
  -C, --no-cache                      disable file caches
  -t, --output-format string          output mode [tui,text] (default "tui")
  -q, --quiet                         disable logging of output from actions
  -R, --reuse                         Reuse containers between executions
  -V, --verbose                       verbose output
  -v, --version                       version for ccr
  -f, --workflow-file string          path to workflow to run
  -w, --working-dir string            directory to run workflow against (default ".")

Use "ccr [command] --help" for more information about a command.
```

## Local Development

To build `ccr` locally, you first need to ensure you have *Go* installed. For macos run: `brew install go`

Install golangci-lint, run: `brew install golangci-lint`

Install `ccr` locally by cloning this repo and then running `make install`:

To run all tests, run: `make`

If you want to make changes to `ccr` and test locally, you can run against source with: `go run main.go`

## Contributing

See [development](docs/development.md) documentation to get started.

## Security

See [CONTRIBUTING](CONTRIBUTING.md#security-issue-notifications) for more information.

## License

This project is licensed under the Apache-2.0 License.

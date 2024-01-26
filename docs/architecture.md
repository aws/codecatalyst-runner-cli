# Architecture

This repository contains the source code for the `ccr` CLI. This tool allow application developers to run CodeCatalyst workflows locally.

## Project Layout

At a high level, this repository consists of [Go workspace](https://go.dev/blog/get-familiar-with-workspaces) with three [modules](https://go.dev/blog/using-go-modules). Each module follows the [Standard Go Project Layout](https://github.com/golang-standards/project-layout).

* [codecatalyst-runner/](../codecatalyst-runner) - the code to define [actions](definition.md#action) with CodeCatalyst ADK as well as the code for the `ccr` executable to run CodeCatalyst workflows locally
  * [main.go](../codecatalyst-runner/main.go) - the main entrypoint for the CLI. The arguments are processed by code in `cmd/` package.
  * [cmd/](../codecatalyst-runner/cmd) - code that defines the arguments and subcommands for the CLI using the [Cobra](https://cobra.dev/) CLI framework. Each subcommand (e.g. `ccr execute`) is defined in a separate file. The command in `cmd/` should be lightweight and delegate actual work to the `pkg/` packages.
  * [pkg/](../codecatalyst-runner/pkg) - code that runs [actions](definitions.md#actions). This code is used by the CLI.
    * [actions/](../codecatalyst-runner/pkg/actions) - code to load [actions](./definitions.md#action) from local paths or remote URLs. Also contains implementations of the [Plan](#plan) interface that is defined from the action. Defines[features](#feature) to run actions such as input/output handlers.
    * [workflows/](../codecatalyst-runner/pkg/workflows) - code to load CodeCatalyst workflows from local paths. Also contains implementations of the [Plan](#plan) interface that is defined from the workflow. Defines [feature](#feature) to run workflows such as report processors, variable handlers, and file caching.

* [command-runner/](../command-runner) - the code to plan and execute commands in Docker, Finch or the local shell.
  * [internal/](../command-runner/internal) - code that supports the `pkg/` packages but is not available to be called outside the `command-runner` module. This package includes utilities to work with tar files, zip files, JSONL files, and git repositories.
  * [pkg/](../command-runner/pkg) - code that defines plans and runs them as commands in Docker, Finch, or the local shell. This code is used by the `codecatalyst-runner`.
    * [common/](../command-runner/pkg/common) - code that implements the [executor](#executor) pattern.
    * [features/](../command-runner/pkg/features) - generic [features](#feature) that aren't tied to CodeCatalyst specific capabilities. Examples include loggers, working directory importers, and plan dependencies.
    * [runner/](../command-runner/pkg/runner) - code that defines the [Plan](#plan) and [Feature](#feature) interfaces. Also contains code to implement Finch, Docker, and Shell plan runners.

## Patterns

The following patterns are found throughout this repository.

### Plan

The [Plan](../command-runner/pkg/runner/plan.go) interface describes a set of commands to run along with the environment to be used for running those commands. This abstraction allows plans to be defined in various forms (e.g. CodeCatalyst workflows) and run consistently. Additionally, the abstraction allows different runner to be implemented (e.g. Finch, Docker, and Shell).

### Feature

The [Feature](../command-runner/pkg/runner/plan.go) type describes a function that wraps the running of a plan. This construct allows new features to be developed and applied during the execution of Plan. These features can be developed, tested, and applied indepenendently. Features have the ability to update the Plan before being run.

### Executor

The executor pattern is defined in [command-runner/pkg/common/executor.go](../command-runner/pkg/common/executor.go). An `Executor` is a functino that receives a `Context` and optionally returns an error. This pattern enables a functional style of programming where pipelines of `Executor`s are constructed to run in series or in parallel. Errors can be caught through `Catch()`.

```go
type Executor func(ctx context.Context) error
```

### Dependency Injection

Dependencies ought to be passed into functions, rather than instantiated within functions. For example, a session from AWS S3 SDK should not be initialized within a function that uses it. Instead, pass an [Interface](#interfaces-for-testing) as a [Struct Param](#struct-params) to the function. This approach improves composability and testability of the code.

### Struct Params

The standard signature should use a `Context` for the first parameter and a struct for the second. The struct should be specifically defined for the function being called with the same name plus a `Params` suffix.

```go

type MyFuncParams struct {
  Foo string
  Bar string
}

func MyFunc(ctx context.Context, params MyFuncParams) error {
  ....
}

```

### Interfaces for Testing

Functions should define their own interfaces for the parameters they take. This allows unit tests to be created that mock these interfaces. The example below demonstrates a function that depends on the AWS S3 SDK. Rather than pass the entire S3 client, an interface is defined with just the functions that the`MyFunc()` uses. This allows us to write unit tests for `MyFunc()` and we only have to mock the two functions from the S3 SDK that are used.

```go
type MyFuncParams struct {
  Foo      string
  Bar      string
  S3Client MyObjectAPIClient
}

type MyObjectAPIClient interface {
  GetObject(context.Context, *s3.GetObjectInput, ...func(*s3.Options)) (*s3.GetObjectOutput, error)
  PutObject(context.Context, *s3.PutObjectInput, ...func(*s3.Options)) (*s3.PutObjectOutput, error)
}

func MyFunc(ctx context.Context, params MyFuncParams) error {
  ....
}
```

### Table Driven Tests

Tests are kept close the the code under test. Create a file of the same name as the code under test with a `_test.go` suffix in the same directory as the code under test. Use [table driven tests](https://dave.cheney.net/2019/05/07/prefer-table-driven-tests) to reduce duplication in tests.

### Go Doc Comments

All types and functions that are available outside their package (start with an upper case) must be documented with [go doc comments](https://tip.golang.org/doc/comment).

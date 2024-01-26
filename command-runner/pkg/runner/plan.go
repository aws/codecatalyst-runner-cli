package runner

import (
	"context"
	"io"
)

// Plan describes how the [Runner] should run
type Plan interface {
	EnvironmentConfiguration() *EnvironmentConfiguration
	CommandGroups() []*CommandGroup
	ID() string
	DependsOn() []string
	AddDependsOn(...string)
}

// Feature applies a feature to a run plan before and/or after the execution
type Feature func(ctx context.Context, plan Plan, execution PlanExecutor) error

// PlanExecutor describes the execution argument passed to a Feature
type PlanExecutor func(ctx context.Context) error

// EnvironmentConfiguration describes an environment to run commands in
type EnvironmentConfiguration struct {
	Env        map[string]string // map of environment variables to set
	Stdout     io.Writer         // where to send stdout to
	Stderr     io.Writer         // where to send stderr to
	WorkingDir string            // working directory to run commands from
	FileMaps   []*FileMap        // FileMaps to make available to commands
	Reuse      bool              // if true, reuse the same environment between executions
}

// CommandGroup describes how to run a set of [Command]s.
// Commands run in a container if Image or BuildContext is set. Otherwise, commands run in a local shell.
type CommandGroup struct {
	Image      string    // Image to pull and run commands in. Ignored if BuildContext is set.
	Entrypoint Command   // Entrypoint to run in container. Only used if BuildContext or Image is set.
	Commands   []Command // Commands to run
}

// Command contains a list of arguments for a given command
type Command []string

// FileMapType describes the type of FileMap. Valid values are copy and mount.
type FileMapType string

const (
	// FileMapTypeCopyInWithGitignore copies files, excluding the .gitignore files
	FileMapTypeCopyInWithGitignore FileMapType = "copy_in_with_gitignore"
	// FileMapTypeCopyIn copies files
	FileMapTypeCopyIn FileMapType = "copy_in"
	// FileMapTypeBind mounts a directory
	FileMapTypeBind FileMapType = "bind"
	// FileMapTypeCopyOut copies files out of a container
	FileMapTypeCopyOut FileMapType = "copy_out"
)

// FileMap describes a mapping between a source path and a target path in the command runner
type FileMap struct {
	SourcePath string //
	TargetPath string
	Type       FileMapType
}

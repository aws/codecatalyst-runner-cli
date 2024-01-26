package types

import (
	"context"
	"io"

	"github.com/aws/codecatalyst-runner-cli/command-runner/pkg/common"
)

// ContainerServiceProvider describes how a service can be created
type ContainerServiceProvider interface {
	Available(ctx context.Context) bool
	NewContainerService() ContainerService
}

type ContainerService interface {
	NewContainer(input NewContainerInput) Container
	ImageExistsLocally(ctx context.Context, imageName string, platform string) (bool, error)
	BuildImage(input BuildImageInput) common.Executor
}

// NewContainerInput the input for the New function
type NewContainerInput struct {
	Image       string
	Username    string
	Password    string
	Entrypoint  []string
	Cmd         []string
	WorkingDir  string
	Env         []string
	Binds       []string
	Mounts      map[string]string
	Name        string
	Stdout      io.Writer
	Stderr      io.Writer
	NetworkMode string
	Privileged  bool
	UsernsMode  string
	Platform    string
	Options     string
}

type BuildImageInput struct {
	ContextDir string
	Dockerfile string
	ImageTag   string
	Platform   string
}

// FileEntry is a file to copy to a container
type FileEntry struct {
	Name string
	Mode int64
	Body string
}

// Container for managing docker run containers
type Container interface {
	Create(capAdd []string, capDrop []string) common.Executor
	CopyIn(containerPath string, hostPath string, useGitIgnore bool) common.Executor
	CopyOut(hostPath string, containerPath string) common.Executor
	Pull(forcePull bool) common.Executor
	Start(attach bool) common.Executor
	Exec(command []string, env map[string]string, user, workdir string) common.Executor
	Remove() common.Executor
	Close() common.Executor
}

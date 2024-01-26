package runner

import (
	"context"
	"fmt"

	"github.com/aws/codecatalyst-runner-cli/command-runner/internal/containers"
	"github.com/aws/codecatalyst-runner-cli/command-runner/internal/containers/types"
)

type commandExecutor interface {
	ExecuteCommand(ctx context.Context, command Command) error
	Close(isError bool) error
}

func newCommandExecutor(ctx context.Context, id string, executionType ExecutionType, commandGroup *CommandGroup, environmentConfiguration *EnvironmentConfiguration) (commandExecutor, error) {
	var containerServiceProvider types.ContainerServiceProvider
	switch executionType {
	case ExecutionTypeShell:
		return newShellCommandExecutor(ctx, &newShellCommandExecutorParams{
			EnvironmentConfiguration: environmentConfiguration,
		})
	case ExecutionTypeDocker:
		containerServiceProvider = containers.Docker
	case ExecutionTypeFinch:
		containerServiceProvider = containers.Finch
	default:
		containerServiceProvider = containers.DefaultServiceProvider(ctx)
	}
	if !containerServiceProvider.Available(ctx) {
		return nil, fmt.Errorf("service provider is unavailable: %T", containerServiceProvider)
	}
	return newContainerCommandExecutor(ctx, &newContainerCommandExecutorParams{
		ID:                       id,
		Image:                    commandGroup.Image,
		Entrypoint:               commandGroup.Entrypoint,
		EnvironmentConfiguration: environmentConfiguration,
		ContainerService:         containerServiceProvider.NewContainerService(),
	})
}

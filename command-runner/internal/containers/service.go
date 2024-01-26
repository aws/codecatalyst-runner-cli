package containers

import (
	"context"

	"github.com/aws/codecatalyst-runner-cli/command-runner/internal/containers/docker"
	"github.com/aws/codecatalyst-runner-cli/command-runner/internal/containers/finch"
	"github.com/aws/codecatalyst-runner-cli/command-runner/internal/containers/types"
)

// Finch service provider
var Finch = &finch.ServiceProvider{}

// Docker service provider
var Docker = &docker.ServiceProvider{}

var providers = []types.ContainerServiceProvider{
	Finch,
	Docker,
}

// DefaultServiceProvider uses the first available service provider
func DefaultServiceProvider(ctx context.Context) types.ContainerServiceProvider {
	for _, provider := range providers {
		if provider.Available(ctx) {
			return provider
		}
	}

	return nil
}

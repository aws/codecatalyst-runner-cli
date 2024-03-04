package docker

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/aws/codecatalyst-runner-cli/command-runner/pkg/common"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/config"
	"github.com/docker/cli/cli/connhelper"
	"github.com/docker/cli/cli/context/docker"
	"github.com/docker/cli/cli/context/store"
	cliflags "github.com/docker/cli/cli/flags"
	"github.com/docker/docker/client"
	"github.com/rs/zerolog/log"
)

// getDockerEndpointOpts results the endpoint for current docker context
func getDockerEndpointOpts(ctx context.Context) ([]client.Opt, error) {
	opts := &cliflags.ClientOptions{}
	storeConfig := command.DefaultContextStoreConfig()
	contextStore := &command.ContextStoreWithDefault{
		Store: store.New(config.ContextStoreDir(), storeConfig),
		Resolver: func() (*command.DefaultContext, error) {
			return command.ResolveDefaultContext(opts, storeConfig)
		},
	}
	endpoint, err := resolveDockerEndpoint(ctx, contextStore, resolveContextName())
	if err != nil {
		return nil, fmt.Errorf("unable to resolve docker endpoint: %w", err)
	}
	return endpoint.ClientOpts()
}

// resolveDockerEndpoint from store
func resolveDockerEndpoint(ctx context.Context, s store.Reader, contextName string) (docker.Endpoint, error) {
	if s == nil {
		return docker.Endpoint{}, fmt.Errorf("no context store initialized")
	}
	log.Ctx(ctx).Debug().Msgf("Using docker context '%s'", contextName)
	ctxMeta, err := s.GetMetadata(contextName)
	if err != nil {
		return docker.Endpoint{}, err
	}
	epMeta, err := docker.EndpointFromContext(ctxMeta)
	if err != nil {
		return docker.Endpoint{}, err
	}
	log.Ctx(ctx).Debug().Msgf("%sdocker context=%s host=%s", logPrefix, contextName, epMeta.Host)
	return docker.WithTLSData(s, contextName, epMeta)
}

// resolveContextName from environment variable or use current
func resolveContextName() string {
	cfg := config.LoadDefaultConfigFile(os.Stderr)
	if os.Getenv(client.EnvOverrideHost) != "" {
		return command.DefaultContextName
	}
	if ctxName := os.Getenv(command.EnvOverrideContext); ctxName != "" {
		return ctxName
	}
	if cfg != nil && cfg.CurrentContext != "" {
		// We don't validate if this context exists: errors may occur when trying to use it.
		return cfg.CurrentContext
	}
	return command.DefaultContextName
}

// getDockerClient returns the Docker APIClient
func getDockerClient(ctx context.Context) (cli client.APIClient, err error) {
	opts, err := getDockerEndpointOpts(ctx)
	if err != nil {
		return nil, err
	}
	dockerHost := os.Getenv("DOCKER_HOST")

	if strings.HasPrefix(dockerHost, "ssh://") {
		var helper *connhelper.ConnectionHelper

		helper, err = connhelper.GetConnectionHelper(dockerHost)
		if err != nil {
			return nil, err
		}
		cli, err = client.NewClientWithOpts(
			client.WithHost(helper.Host),
			client.WithDialContext(helper.Dialer),
		)
	} else {
		cli, err = client.NewClientWithOpts(opts...)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to connect to docker daemon: %w", err)
	}
	cli.NegotiateAPIVersion(ctx)

	return cli, nil
}

func (cr *dockerContainer) connect() common.Executor {
	return func(ctx context.Context) error {
		if cr.cli != nil {
			return nil
		}
		cli, err := getDockerClient(ctx)
		if err != nil {
			return err
		}
		cr.cli = cli
		return nil
	}
}

//go:build !(WITHOUT_DOCKER || !(linux || darwin || windows))

package docker

import (
	"context"
	"strings"

	"github.com/docker/cli/cli/config"
	"github.com/docker/cli/cli/config/credentials"
	"github.com/docker/docker/api/types/registry"
	"github.com/rs/zerolog/log"
)

// loadDockerAuthConfig loads the docker auth config for the given image
func loadDockerAuthConfig(ctx context.Context, image string) (registry.AuthConfig, error) {
	config, err := config.Load(config.Dir())
	if err != nil {
		log.Ctx(ctx).Warn().Msgf("Could not load docker config: %v", err)
		return registry.AuthConfig{}, err
	}

	if !config.ContainsAuth() {
		config.CredentialsStore = credentials.DetectDefaultStore(config.CredentialsStore)
	}

	hostName := "index.docker.io"
	index := strings.IndexRune(image, '/')
	if index > -1 && (strings.ContainsAny(image[:index], ".:") || image[:index] == "localhost") {
		hostName = image[:index]
	}

	authConfig, err := config.GetAuthConfig(hostName)
	if err != nil {
		log.Ctx(ctx).Warn().Msgf("Could not get auth config from docker config: %v", err)
		return registry.AuthConfig{}, err
	}

	return registry.AuthConfig(authConfig), nil
}

// loadDockerAuthConfigs loads all docker credentials
func loadDockerAuthConfigs(ctx context.Context) map[string]registry.AuthConfig {
	config, err := config.Load(config.Dir())
	if err != nil {
		log.Ctx(ctx).Warn().Msgf("Could not load docker config: %v", err)
		return nil
	}

	if !config.ContainsAuth() {
		config.CredentialsStore = credentials.DetectDefaultStore(config.CredentialsStore)
	}

	creds, _ := config.GetAllCredentials()
	authConfigs := make(map[string]registry.AuthConfig, len(creds))
	for k, v := range creds {
		authConfigs[k] = registry.AuthConfig(v)
	}

	return authConfigs
}

//go:build !(WITHOUT_DOCKER || !(linux || darwin || windows))

package docker

import (
	"context"
	"fmt"
	"os"

	"github.com/aws/codecatalyst-runner-cli/command-runner/internal/containers/types"

	"github.com/docker/docker/client"
	"github.com/rs/zerolog/log"
)

type ServiceProvider struct{}

func (sp *ServiceProvider) Available(ctx context.Context) bool {
	if os.Getenv("NODOCKER") != "" {
		return false
	}
	cli, err := getDockerClient(ctx)
	if err != nil {
		log.Debug().Err(err).Msg("docker is unavailable")
		return false
	}
	defer cli.Close()
	_, err = cli.ServerVersion(ctx)
	if err != nil {
		log.Debug().Err(err).Msg("docker is unavailable")
		return false
	}
	return true
}

func (sp *ServiceProvider) NewContainerService() types.ContainerService {
	return &dockerContainerService{}
}

type dockerContainerService struct{}

func (dcs *dockerContainerService) NewContainer(input types.NewContainerInput) types.Container {
	cr := new(dockerContainer)
	cr.input = input
	return cr
}

// ImageExistsLocally returns a boolean indicating if an image with the
// requested name, tag and architecture exists in the local docker image store
func (dcs *dockerContainerService) ImageExistsLocally(ctx context.Context, imageName string, platform string) (bool, error) {
	return imageExistsLocally(ctx, imageName, platform)
}

func imageExistsLocally(ctx context.Context, imageName string, platform string) (bool, error) {
	cli, err := getDockerClient(ctx)
	if err != nil {
		return false, err
	}
	defer cli.Close()

	inspectImage, _, err := cli.ImageInspectWithRaw(ctx, imageName)
	if client.IsErrNotFound(err) {
		return false, nil
	} else if err != nil {
		return false, err
	}

	if platform == "" || platform == "any" || fmt.Sprintf("%s/%s", inspectImage.Os, inspectImage.Architecture) == platform {
		return true, nil
	}

	return false, nil
}

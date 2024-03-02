package docker

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/aws/codecatalyst-runner-cli/command-runner/pkg/common"

	"github.com/distribution/reference"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/registry"
	"github.com/rs/zerolog/log"
)

// newDockerPullExecutorInput the input for the NewDockerPullExecutor function
type newDockerPullExecutorInput struct {
	Image     string
	ForcePull bool
	Platform  string
	Username  string
	Password  string
}

// newDockerPullExecutor function to create a run executor for the container
func newDockerPullExecutor(input newDockerPullExecutorInput) common.Executor {
	return func(ctx context.Context) error {
		log.Ctx(ctx).Printf("%sdocker pull %v", logPrefix, input.Image)

		if common.Dryrun(ctx) {
			return nil
		}

		pull := input.ForcePull
		if !pull {
			imageExists, err := imageExistsLocally(ctx, input.Image, input.Platform)
			log.Ctx(ctx).Printf("Image exists? %v", imageExists)
			if err != nil {
				return fmt.Errorf("unable to determine if image already exists for image '%s' (%s): %w", input.Image, input.Platform, err)
			}

			if !imageExists {
				pull = true
			}
		}

		if !pull {
			return nil
		}

		imageRef := cleanImage(ctx, input.Image)
		log.Ctx(ctx).Printf("pulling image '%v' (%s)", imageRef, input.Platform)

		cli, err := getDockerClient(ctx)
		if err != nil {
			return err
		}
		defer cli.Close()

		imagePullOptions, err := getImagePullOptions(ctx, input)
		if err != nil {
			return err
		}

		reader, err := cli.ImagePull(ctx, imageRef, imagePullOptions)

		_ = logDockerResponse(log.Ctx(ctx), reader, err != nil)
		if err != nil {
			return err
		}
		return nil
	}
}

func getImagePullOptions(ctx context.Context, input newDockerPullExecutorInput) (types.ImagePullOptions, error) {
	imagePullOptions := types.ImagePullOptions{
		Platform: input.Platform,
	}

	if input.Username != "" && input.Password != "" {
		log.Ctx(ctx).Printf("using authentication for docker pull")

		authConfig := registry.AuthConfig{
			Username: input.Username,
			Password: input.Password,
		}

		encodedJSON, err := json.Marshal(authConfig)
		if err != nil {
			return imagePullOptions, err
		}

		imagePullOptions.RegistryAuth = base64.URLEncoding.EncodeToString(encodedJSON)
	} else {
		authConfig, err := loadDockerAuthConfig(ctx, input.Image)
		if err != nil {
			return imagePullOptions, err
		}
		if authConfig.Username == "" && authConfig.Password == "" {
			return imagePullOptions, nil
		}

		encodedJSON, err := json.Marshal(authConfig)
		if err != nil {
			return imagePullOptions, err
		}

		imagePullOptions.RegistryAuth = base64.URLEncoding.EncodeToString(encodedJSON)
	}

	return imagePullOptions, nil
}

func cleanImage(ctx context.Context, image string) string {
	ref, err := reference.ParseAnyReference(image)
	if err != nil {
		log.Ctx(ctx).Err(err)
		return ""
	}

	return ref.String()
}

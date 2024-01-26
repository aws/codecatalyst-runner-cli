package finch

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/aws/codecatalyst-runner-cli/command-runner/pkg/common"

	"github.com/rs/zerolog/log"
)

// newPullExecutorInput the input for the NewDockerPullExecutor function
type newPullExecutorInput struct {
	Image     string
	ForcePull bool
	Platform  string
	Username  string
	Password  string
}

// newPullExecutor function to create a run executor for the container
func newPullExecutor(input newPullExecutorInput) common.Executor {
	return func(ctx context.Context) error {
		log.Ctx(ctx).Printf("containerd pull %s force? %v", input.Image, input.ForcePull)

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

		f, err := newFinch(finchInstallDir)
		if err != nil {
			return err
		}

		imageRef := input.Image
		log.Ctx(ctx).Printf("pulling image '%v' (%s)", imageRef, input.Platform)

		_, _, err = f.RunWithoutStdio(ctx, "pull", "-q", imageRef)
		return err
	}
}

type finchImageSpec struct {
	ID           string   `json:"Id"`
	RepoTags     []string `json:"RepoTags"`
	OS           string   `json:"Os"`
	Architecture string   `json:"Architecture"`
}

func imageExistsLocally(ctx context.Context, imageName string, platform string) (bool, error) {
	f, err := newFinch(finchInstallDir)
	if err != nil {
		return false, err
	}

	rout, rerr, err := f.RunWithoutStdio(ctx, "inspect", imageName)
	if err != nil {
		if strings.Contains(string(rerr), "no such object") {
			return false, nil
		}
		return false, fmt.Errorf("unable to inspect image '%s': %w\n%s", imageName, err, rerr)
	}
	images := make([]finchImageSpec, 0)
	if err := json.Unmarshal(rout, &images); err != nil {
		return false, err
	}

	for _, image := range images {
		if platform == "" || platform == "any" || platform == fmt.Sprintf("%s/%s", image.OS, image.Architecture) {
			return true, nil
		}
	}

	return false, nil
}

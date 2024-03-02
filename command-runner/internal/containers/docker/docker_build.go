package docker

import (
	"context"
	"io"
	"os"
	"path/filepath"

	ctypes "github.com/aws/codecatalyst-runner-cli/command-runner/internal/containers/types"
	"github.com/aws/codecatalyst-runner-cli/command-runner/pkg/common"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/pkg/archive"
	"github.com/rs/zerolog/log"

	"github.com/moby/buildkit/frontend/dockerfile/dockerignore"
	"github.com/moby/patternmatcher"
)

// BuildImage function to create a run executor for the container
func (dcs *dockerContainerService) BuildImage(input ctypes.BuildImageInput) common.Executor {
	return func(ctx context.Context) error {
		logger := log.Ctx(ctx)
		if input.Platform != "" {
			logger.Info().Msgf("%sdocker build -t %s --platform %s %s", logPrefix, input.ImageTag, input.Platform, input.ContextDir)
		} else {
			logger.Info().Msgf("%sdocker build -t %s %s", logPrefix, input.ImageTag, input.ContextDir)
		}
		if common.Dryrun(ctx) {
			return nil
		}

		cli, err := getDockerClient(ctx)
		if err != nil {
			return err
		}
		defer cli.Close()

		logger.Debug().Msgf("Building image from '%v'", input.ContextDir)

		tags := []string{input.ImageTag}
		options := types.ImageBuildOptions{
			Tags:        tags,
			Remove:      true,
			Platform:    input.Platform,
			AuthConfigs: loadDockerAuthConfigs(ctx),
			Dockerfile:  input.Dockerfile,
		}
		buildContext, err := createBuildContext(ctx, input.ContextDir, input.Dockerfile)
		if err != nil {
			return err
		}

		defer buildContext.Close()

		logger.Debug().Msgf("Creating image from context dir '%s' with tag '%s' and platform '%s'", input.ContextDir, input.ImageTag, input.Platform)
		resp, err := cli.ImageBuild(ctx, buildContext, options)
		if err != nil {
			return err
		}

		err = logDockerResponse(log.Ctx(ctx), resp.Body, err != nil)
		if err != nil {
			return err
		}
		return nil
	}
}
func createBuildContext(ctx context.Context, contextDir string, relDockerfile string) (io.ReadCloser, error) {
	log.Ctx(ctx).Debug().Msgf("Creating archive for build context dir '%s' with relative dockerfile '%s'", contextDir, relDockerfile)

	// And canonicalize dockerfile name to a platform-independent one
	relDockerfile = archive.CanonicalTarNameForPath(relDockerfile)

	f, err := os.Open(filepath.Join(contextDir, ".dockerignore"))
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	defer f.Close()

	var excludes []string
	if err == nil {
		excludes, err = dockerignore.ReadAll(f)
		if err != nil {
			return nil, err
		}
	}

	// If .dockerignore mentions .dockerignore or the Dockerfile
	// then make sure we send both files over to the daemon
	// because Dockerfile is, obviously, needed no matter what, and
	// .dockerignore is needed to know if either one needs to be
	// removed. The daemon will remove them for us, if needed, after it
	// parses the Dockerfile. Ignore errors here, as they will have been
	// caught by validateContextDirectory above.
	var includes = []string{"."}
	keepThem1, _ := patternmatcher.Matches(".dockerignore", excludes)
	keepThem2, _ := patternmatcher.Matches(relDockerfile, excludes)
	if keepThem1 || keepThem2 {
		includes = append(includes, ".dockerignore", relDockerfile)
	}

	compression := archive.Uncompressed
	buildCtx, err := archive.TarWithOptions(contextDir, &archive.TarOptions{
		Compression:     compression,
		ExcludePatterns: excludes,
		IncludeFiles:    includes,
	})
	if err != nil {
		return nil, err
	}

	return buildCtx, nil
}

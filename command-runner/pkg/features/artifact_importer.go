package features

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/aws/codecatalyst-runner-cli/command-runner/pkg/runner"

	"github.com/rs/zerolog/log"
)

// ArtifactImporter imports artifacts from a directory into the container
func ArtifactImporter(artifactDir string, artifacts ...string) runner.Feature {
	return func(ctx context.Context, plan runner.Plan, e runner.PlanExecutor) error {
		log.Ctx(ctx).Debug().Msg("ENTER ArtifactImporter")
		artifactDir, err := filepath.Abs(artifactDir)
		if err != nil {
			return fmt.Errorf("unable to get abs directory: %w", err)
		}
		envCfg := plan.EnvironmentConfiguration()
		for _, artifact := range artifacts {
			envCfg.FileMaps = append(envCfg.FileMaps, &runner.FileMap{
				SourcePath: filepath.Join(artifactDir, artifact),
				TargetPath: "./",
				Type:       runner.FileMapTypeCopyIn,
			})
		}
		err = e(ctx)
		log.Ctx(ctx).Debug().Msg("EXIT ArtifactImporter")
		return err
	}
}

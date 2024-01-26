package workflows

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/aws/codecatalyst-runner-cli/command-runner/pkg/runner"

	"github.com/rs/zerolog/log"
)

// OutputArtifacts stores artifacts from the execution
func OutputArtifacts(planID string, out []*OutputArtifact, artifactPlans map[string]string, cacheDir string) runner.Feature {
	for _, artifact := range out {
		artifactPlans[artifact.Name] = planID
	}
	return func(ctx context.Context, plan runner.Plan, e runner.PlanExecutor) error {
		log.Ctx(ctx).Debug().Msg("ENTER OutputArtifacts")
		envCfg := plan.EnvironmentConfiguration()
		if envCfg.Env == nil {
			envCfg.Env = make(map[string]string)
		}
		for _, artifact := range out {
			var files []string
			switch typedFiles := artifact.Files.(type) {
			case []interface{}:
				for _, inf := range typedFiles {
					files = append(files, inf.(string))
				}
			case []string:
				files = typedFiles
			case string:
				files = []string{typedFiles}
			default:
				return fmt.Errorf("invalid files type: %T", artifact.Files)
			}

			for _, file := range files {
				fileParts := strings.Split(file, "*")
				targetPath := filepath.Join(cacheDir, "artifacts", artifact.Name, fileParts[0])
				if len(fileParts) > 1 {
					targetPath = fmt.Sprintf("%s/", targetPath)
				}
				envCfg.FileMaps = append(envCfg.FileMaps, &runner.FileMap{
					Type:       runner.FileMapTypeCopyOut,
					SourcePath: filepath.Join("git", "v1", filepath.Base(envCfg.WorkingDir), file),
					TargetPath: targetPath,
				})
			}
		}
		err := e(ctx)

		log.Ctx(ctx).Debug().Msg("EXIT OutputArtifacts")
		return err
	}
}

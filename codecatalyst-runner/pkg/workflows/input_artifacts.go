package workflows

import (
	"context"
	"fmt"
	"path/filepath"
	"slices"

	"github.com/aws/codecatalyst-runner-cli/command-runner/pkg/common"
	"github.com/aws/codecatalyst-runner-cli/command-runner/pkg/runner"

	"github.com/rs/zerolog/log"
)

// InputArtifacts imports artifacts from a directory into the container
func InputArtifacts(in []string, artifactPlans map[string]string, cacheDir string) runner.Feature {
	return func(ctx context.Context, plan runner.Plan, e runner.PlanExecutor) error {
		log.Ctx(ctx).Debug().Msg("ENTER InputArtifacts")
		envCfg := plan.EnvironmentConfiguration()
		if envCfg.Env == nil {
			envCfg.Env = make(map[string]string)
		}
		for i, artifact := range in {
			if depPlanID, ok := artifactPlans[artifact]; !ok {
				log.Ctx(ctx).Debug().Msgf("DEFER - no plan started yet for artifact %s", artifact)
				return common.ErrDefer
			} else if !slices.Contains(plan.DependsOn(), depPlanID) {
				log.Ctx(ctx).Debug().Msgf("DEFER - waiting for plan %s to provide artifact %s", depPlanID, artifact)
				plan.AddDependsOn(depPlanID)
				return common.ErrDefer
			}
			targetPath := fmt.Sprintf("s3/%02d", i+1)
			envCfg.Env[fmt.Sprintf("CATALYST_SOURCE_DIR_%s", artifact)] = targetPath
			envCfg.FileMaps = append(envCfg.FileMaps, &runner.FileMap{
				Type:       runner.FileMapTypeCopyIn,
				SourcePath: fmt.Sprintf("%s/.", filepath.Join(cacheDir, "artifacts", artifact)),
				TargetPath: targetPath,
			})
		}
		log.Ctx(ctx).Debug().Msgf("env:%+v", envCfg.Env)
		err := e(ctx)
		log.Ctx(ctx).Debug().Msg("EXIT InputArtifacts")
		return err
	}
}

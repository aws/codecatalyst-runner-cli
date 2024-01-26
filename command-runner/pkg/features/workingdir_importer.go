package features

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/aws/codecatalyst-runner-cli/command-runner/pkg/runner"

	"github.com/rs/zerolog/log"
)

// WorkingDirImporter is a Feature that creates FileMaps to bind or copy the working directory
// to the environment used to execute a Plan.
func WorkingDirImporter(workingDir string, cacheDir string, bindWorkingDir bool, sharedCompute bool, isWorkingDirSetup *bool) runner.Feature {
	return func(ctx context.Context, plan runner.Plan, e runner.PlanExecutor) error {
		log.Ctx(ctx).Debug().Msgf("ENTER WorkingDirImporter workingDir=%s bind=%v shared=%v", workingDir, bindWorkingDir, sharedCompute)
		workingDir, err := filepath.Abs(workingDir)
		if err != nil {
			return fmt.Errorf("unable to get abs directory: %w", err)
		}
		envCfg := plan.EnvironmentConfiguration()
		targetPath := filepath.Join("git", "v1", filepath.Base(workingDir))
		if bindWorkingDir {
			envCfg.FileMaps = append(envCfg.FileMaps, &runner.FileMap{
				SourcePath: workingDir,
				TargetPath: targetPath,
				Type:       runner.FileMapTypeBind,
			})
		} else if sharedCompute {
			workflowSourceCacheDir := filepath.Join(cacheDir, "sources", "WorkflowSource")
			if !*isWorkingDirSetup {
				envCfg.FileMaps = append(envCfg.FileMaps, &runner.FileMap{
					SourcePath: fmt.Sprintf("%s/.", workingDir),
					TargetPath: targetPath,
					Type:       runner.FileMapTypeCopyInWithGitignore,
				})
				*isWorkingDirSetup = true
			} else {
				envCfg.FileMaps = append(envCfg.FileMaps, &runner.FileMap{
					SourcePath: fmt.Sprintf("%s/.", workflowSourceCacheDir),
					TargetPath: targetPath,
					Type:       runner.FileMapTypeCopyInWithGitignore,
				})
			}
			envCfg.FileMaps = append(envCfg.FileMaps, &runner.FileMap{
				SourcePath: fmt.Sprintf("%s/.", targetPath),
				TargetPath: workflowSourceCacheDir,
				Type:       runner.FileMapTypeCopyOut,
			})
		} else {
			envCfg.FileMaps = append(envCfg.FileMaps, &runner.FileMap{
				SourcePath: fmt.Sprintf("%s/.", workingDir),
				TargetPath: targetPath,
				Type:       runner.FileMapTypeCopyInWithGitignore,
			})
		}
		if envCfg.Env == nil {
			envCfg.Env = make(map[string]string)
		}
		envCfg.Env["CATALYST_SOURCE_DIR_WorkflowSource"] = filepath.Join("git/v1", filepath.Base(workingDir))
		err = e(ctx)
		log.Ctx(ctx).Debug().Msg("EXIT WorkingDirImporter")
		return err
	}
}

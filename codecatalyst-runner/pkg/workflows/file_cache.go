package workflows

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/aws/codecatalyst-runner-cli/command-runner/pkg/runner"

	"github.com/rs/zerolog/log"
)

type FileCacheDirProvider func(ctx context.Context, plan runner.Plan) (string, error)

// FileCache paths within actons
func FileCache(workingDir string, fileCaching FileCaching, cacheDirProvider FileCacheDirProvider) runner.Feature {
	return func(ctx context.Context, plan runner.Plan, e runner.PlanExecutor) error {
		log.Ctx(ctx).Debug().Msg("ENTER FileCache")

		cacheDir, err := cacheDirProvider(ctx, plan)
		if err != nil {
			return err
		}

		envCfg := plan.EnvironmentConfiguration()
		for key, entry := range fileCaching {
			cachePath := fmt.Sprintf("%s%s", filepath.Join(cacheDir, "caches", key), string(filepath.Separator))
			containerWorkingDir := filepath.Join("git", "v1", filepath.Base(workingDir))
			var hostSourcePath, hostTargetPath, containerSourcePath, containerTargetPath string
			hostSourcePath = fmt.Sprintf("%s.", cachePath)
			hostTargetPath = cachePath
			if !filepath.IsAbs(entry.Path) {
				containerSourcePath = filepath.Join(containerWorkingDir, entry.Path)
			} else {
				containerSourcePath = entry.Path
			}
			containerTargetPath = filepath.Dir(containerSourcePath) + "/"

			var isBound bool
			if _, err := os.Stat(cachePath); err == nil {
				log.Ctx(ctx).Debug().Msgf("📦 Restoring from cache %s", cachePath)
				if path, err := os.Stat(filepath.Join(cachePath, filepath.Base(entry.Path))); err == nil && path.IsDir() {
					envCfg.FileMaps = append(envCfg.FileMaps, &runner.FileMap{
						Type:       runner.FileMapTypeBind,
						SourcePath: filepath.Join(cachePath, filepath.Base(entry.Path)),
						TargetPath: containerSourcePath,
					})
					isBound = true
				} else {
					envCfg.FileMaps = append(envCfg.FileMaps, &runner.FileMap{
						Type:       runner.FileMapTypeCopyIn,
						SourcePath: hostSourcePath,
						TargetPath: containerTargetPath,
					})
				}
			} else if !os.IsNotExist(err) {
				return err
			}
			if !isBound {
				envCfg.FileMaps = append(envCfg.FileMaps, &runner.FileMap{
					Type:       runner.FileMapTypeCopyOut,
					SourcePath: containerSourcePath,
					TargetPath: hostTargetPath,
				})
			}
		}
		err = e(ctx)
		log.Ctx(ctx).Debug().Msg("EXIT FileCache")
		return err
	}
}

func staticCacheDirProvider(dir string) FileCacheDirProvider {
	return func(ctx context.Context, plan runner.Plan) (string, error) {
		return dir, nil
	}
}

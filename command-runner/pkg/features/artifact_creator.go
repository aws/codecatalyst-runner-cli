package features

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/aws/codecatalyst-runner-cli/command-runner/pkg/common"
	"github.com/aws/codecatalyst-runner-cli/command-runner/pkg/runner"

	"github.com/rs/zerolog/log"
)

// ArtifactCreator zip up an artifact with all files added to the output directory from the action
func ArtifactCreator(artifactDir string, bindToPath string, envToSet string) runner.Feature {
	return func(ctx context.Context, plan runner.Plan, e runner.PlanExecutor) error {
		log.Ctx(ctx).Debug().Msg("ENTER ArtifactCreator")
		if err := os.MkdirAll(artifactDir, 0755); err != nil {
			return err
		}
		envCfg := plan.EnvironmentConfiguration()
		if envCfg.Env == nil {
			envCfg.Env = make(map[string]string)
		}
		if envToSet != "" {
			if bindToPath != "" {
				envCfg.Env[envToSet] = bindToPath
			} else {
				envCfg.Env[envToSet] = artifactDir
			}
		}
		if bindToPath != "" {
			envCfg.FileMaps = append(envCfg.FileMaps, &runner.FileMap{
				SourcePath: artifactDir,
				TargetPath: bindToPath,
				Type:       runner.FileMapTypeBind,
			})
		}

		err := e(ctx)
		if zipErr := newArtifactZipper(artifactDir)(ctx); zipErr != nil {
			log.Ctx(ctx).Error().Err(zipErr).Msgf("unable to zip artifact")
		}
		log.Ctx(ctx).Debug().Msg("EXIT ArtifactCreator")
		return err
	}
}

func newArtifactZipper(artifactDir string) common.Executor {
	zipPath := filepath.Join(filepath.Dir(artifactDir), fmt.Sprintf("%s.zip", filepath.Base(artifactDir)))
	return func(ctx context.Context) error {
		dstFile, err := os.Create(zipPath)
		if err != nil {
			return err
		}
		defer dstFile.Close()

		zipWriter := zip.NewWriter(dstFile)
		defer zipWriter.Close()

		count := 0
		walker := func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				return nil
			}
			src, err := os.Open(path)
			if err != nil {
				return err
			}
			defer src.Close()

			relPath, err := filepath.Rel(artifactDir, path)
			if err != nil {
				return err
			}
			dst, err := zipWriter.Create(relPath)
			if err != nil {
				return err
			}

			_, err = io.Copy(dst, src)
			if err != nil {
				return err
			}
			count++

			return nil
		}
		err = filepath.Walk(artifactDir, walker)
		if err != nil {
			return err
		}
		if count == 0 {
			os.Remove(zipPath)
		}
		return nil
	}
}

package runner

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/aws/codecatalyst-runner-cli/command-runner/internal/fs"
	"github.com/aws/codecatalyst-runner-cli/command-runner/pkg/common"

	"github.com/go-git/go-billy/v5/helper/polyfill"
	"github.com/go-git/go-billy/v5/osfs"
	"github.com/go-git/go-git/v5/plumbing/format/gitignore"
	"github.com/rs/zerolog/log"
)

type shellCommandExecutor struct {
	Env        []string
	Stdout     io.Writer
	Stderr     io.Writer
	WorkingDir string
	CleanupDir string
}

type newShellCommandExecutorParams struct {
	*EnvironmentConfiguration
}

func newShellCommandExecutor(ctx context.Context, params *newShellCommandExecutorParams) (commandExecutor, error) {
	execWorkingDir := params.WorkingDir
	var cleanupDir string
	for _, filemap := range params.FileMaps {
		// only handle filemap for working directory in first pass
		if resolvePath(filemap.SourcePath, params.WorkingDir) == resolvePath(".", params.WorkingDir) {
			if filemap.Type == FileMapTypeCopyIn || filemap.Type == FileMapTypeCopyInWithGitignore {
				// create temporary directory for the working directory
				var err error
				execWorkingDir, err = createCleanWorkdir(ctx, execWorkingDir)
				if err != nil {
					return nil, err
				}
				cleanupDir = filepath.Dir(execWorkingDir)
			}
		}
	}
	for _, filemap := range params.FileMaps {
		switch filemap.Type {
		case FileMapTypeCopyOut:
			if err := copyDir(
				ctx,
				resolvePath(filemap.SourcePath, params.WorkingDir),
				resolvePath(filemap.TargetPath, execWorkingDir),
				false,
			); err != nil {
				return nil, err
			}
		case FileMapTypeBind:
			if resolvePath(filemap.SourcePath, params.WorkingDir) != resolvePath(".", params.WorkingDir) {
				return nil, fmt.Errorf("unable to use bind mounts with shell executor for non-working directory '%s'", filemap.SourcePath)
			}
		case FileMapTypeCopyIn:
			if err := copyDir(
				ctx,
				resolvePath(filemap.TargetPath, execWorkingDir),
				resolvePath(filemap.SourcePath, params.WorkingDir),
				false,
			); err != nil {
				return nil, err
			}
		case FileMapTypeCopyInWithGitignore:
			if err := copyDir(
				ctx,
				resolvePath(filemap.TargetPath, execWorkingDir),
				resolvePath(filemap.SourcePath, params.WorkingDir),
				true,
			); err != nil {
				return nil, err
			}
		default:
			return nil, fmt.Errorf("unknown filemap Type")
		}
	}

	env := make([]string, 0)
	if params.Env != nil {
		for k, v := range params.Env {
			env = append(env, fmt.Sprintf("%s=%s", k, v))
		}
	}
	env = append(env, fmt.Sprintf("PATH=%s", os.Getenv("PATH")))
	env = append(env, fmt.Sprintf("CATALYST_DEFAULT_DIR=%s", execWorkingDir))

	return &shellCommandExecutor{
		Env:        env,
		Stdout:     params.Stdout,
		Stderr:     params.Stderr,
		WorkingDir: execWorkingDir,
		CleanupDir: cleanupDir,
	}, nil
}

func (sce *shellCommandExecutor) Close(_ bool) error {
	if sce.CleanupDir != "" {
		log.Debug().Msgf("close() is removing %s", sce.CleanupDir)
		return os.RemoveAll(sce.CleanupDir)
	}
	return nil
}

func (sce *shellCommandExecutor) ExecuteCommand(ctx context.Context, command Command) error {
	shell := []string{"/bin/bash", "-c"}

	args := shell
	args = append(args, strings.Join(command, " "))
	cmd := exec.CommandContext(ctx, shell[0]) //#nosec G204
	cmd.Path = shell[0]
	cmd.Args = args
	cmd.Stdin = nil
	cmd.Dir = sce.WorkingDir
	cmd.Env = sce.Env

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}

	log.Ctx(ctx).Debug().Msgf("%s shell run command=%+v workdir=%s", logPrefix, cmd, sce.WorkingDir)
	if common.Dryrun(ctx) {
		log.Ctx(ctx).Debug().Msgf("exit for dryrun")
		return nil
	}

	if err := cmd.Start(); err != nil {
		return err
	}
	if sce.Stdout != nil {
		go streamPipe(sce.Stdout, stdout)
	}
	if sce.Stderr != nil {
		go streamPipe(sce.Stderr, stderr)
	}
	return cmd.Wait()
}

const logPrefix = "  \U0001F4BB  "

func streamPipe(dst io.Writer, src io.ReadCloser) {
	reader := bufio.NewReader(src)
	_, _ = io.Copy(dst, reader)
}

func createCleanWorkdir(ctx context.Context, workdir string) (string, error) {
	tmpdir, err := os.MkdirTemp(fs.TmpDir(), "executor-shell")
	if err != nil {
		return "", err
	}
	log.Ctx(ctx).Debug().Msgf("Created clean working directory %s", tmpdir)
	return filepath.Join(tmpdir, filepath.Base(workdir)), nil
}

func copyDir(ctx context.Context, destdir string, sourcedir string, useGitIgnore bool) error {
	if sourcedir == destdir {
		return fmt.Errorf("unable to copyDir when sourcedir==destdir")
	}
	log.Ctx(ctx).Debug().Msgf("Copying from %s to %s", sourcedir, destdir)
	srcPrefix := filepath.Dir(sourcedir)
	if !strings.HasSuffix(srcPrefix, string(filepath.Separator)) {
		srcPrefix += string(filepath.Separator)
	}
	log.Ctx(ctx).Debug().Msgf("Stripping prefix:%s src:%s", srcPrefix, sourcedir)
	var ignorer gitignore.Matcher
	if useGitIgnore {
		ps, err := gitignore.ReadPatterns(polyfill.New(osfs.New(sourcedir)), nil)
		if err != nil {
			log.Ctx(ctx).Debug().Msgf("Error loading .gitignore: %v", err)
		}

		ignorer = gitignore.NewMatcher(ps)
	}
	fc := &fs.FileCollector{
		Fs:        &fs.DefaultFs{},
		Ignorer:   ignorer,
		SrcPath:   sourcedir,
		SrcPrefix: srcPrefix,
		Handler: &fs.CopyCollector{
			DstDir: destdir,
		},
	}
	return filepath.Walk(sourcedir, fc.CollectFiles(ctx, []string{}))
}

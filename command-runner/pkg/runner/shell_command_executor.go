package runner

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/aws/codecatalyst-runner-cli/command-runner/internal/fs"
	"github.com/aws/codecatalyst-runner-cli/command-runner/pkg/common"

	"github.com/go-git/go-billy/v5/helper/polyfill"
	"github.com/go-git/go-billy/v5/osfs"
	"github.com/go-git/go-git/v5/plumbing/format/gitignore"
	"github.com/rs/zerolog/log"
)

type shellCommandExecutor struct {
	Stdout         io.Writer
	Stderr         io.Writer
	Env            []string
	WorkingDir     string
	CloseExecutors []common.Executor
	ctx            context.Context
	mceDir         string
}

type newShellCommandExecutorParams struct {
	*EnvironmentConfiguration
}

func newShellCommandExecutor(ctx context.Context, params *newShellCommandExecutorParams) (commandExecutor, error) {
	mceDir, err := os.MkdirTemp(fs.TmpDir(), "mce")
	if err != nil {
		return nil, err
	}
	closeExecutors := []common.Executor{}
	for _, filemap := range params.FileMaps {
		switch filemap.Type {
		case FileMapTypeCopyOut:
			closeExecutors = append(closeExecutors, copyOut(ctx, mceDir, params.WorkingDir, filemap))
		case FileMapTypeBind:
			// treat bind mount as symlink
			if err := symlink(mceDir, params.WorkingDir, filemap); err != nil {
				return nil, err
			}
		case FileMapTypeCopyIn:
			if err := copyDir(
				ctx,
				resolvePath(filemap.TargetPath, mceDir),
				resolvePath(filemap.SourcePath, params.WorkingDir),
				false,
			); err != nil {
				return nil, err
			}
		case FileMapTypeCopyInWithGitignore:
			if err := copyDir(
				ctx,
				resolvePath(filemap.TargetPath, mceDir),
				resolvePath(filemap.SourcePath, params.WorkingDir),
				true,
			); err != nil {
				return nil, err
			}
		default:
			return nil, fmt.Errorf("unknown filemap Type")
		}
	}

	closeExecutors = append(closeExecutors, setOutputs(mceDir, params.Stdout))
	closeExecutors = append(closeExecutors, func(ctx context.Context) error {
		log.Debug().Msgf("close() is removing %s", mceDir)
		return os.RemoveAll(mceDir)
	})

	env := make([]string, 0)
	var defaultDir string
	if params.Env != nil {
		for k, v := range params.Env {
			if strings.HasPrefix(k, "CATALYST_SOURCE_DIR_") {
				v = resolvePath(v, mceDir)
				if k == "CATALYST_SOURCE_DIR_WorkflowSource" || defaultDir == "" {
					defaultDir = v
				}
			}
			env = append(env, fmt.Sprintf("%s=%s", k, interpolate(v, params.Env)))
		}
	}
	if defaultDir == "" {
		defaultDir = params.WorkingDir
	}
	env = append(env, fmt.Sprintf("PATH=%s", os.Getenv("PATH")))
	env = append(env, fmt.Sprintf("CATALYST_DEFAULT_DIR=%s", defaultDir))

	if err := os.WriteFile(filepath.Join(mceDir, "env.sh"), []byte(""), 00666); err != nil /* #nosec G306 */ {
		return nil, err
	}

	if err := os.WriteFile(filepath.Join(mceDir, "dir.txt"), []byte(defaultDir), 00644); err != nil /* #nosec G306 */ {
		return nil, err
	}

	return &shellCommandExecutor{
		Stdout:         params.Stdout,
		Stderr:         params.Stderr,
		WorkingDir:     defaultDir,
		Env:            env,
		CloseExecutors: closeExecutors,
		mceDir:         mceDir,
		ctx:            ctx,
	}, nil
}

func (sce *shellCommandExecutor) Close(isError bool) error {
	var err error
	if !isError {
		err = common.NewPipelineExecutor(sce.CloseExecutors...).TraceRegion("close-executors")(sce.ctx)
	}
	return err
}

func (sce *shellCommandExecutor) ExecuteCommand(ctx context.Context, command Command) error {
	script := fmt.Sprintf(`
	MCE_DIR=%s
	cd $(cat ${MCE_DIR}/dir.txt)
	set -a
	. ${MCE_DIR}/env.sh
	%s
	CODEBUILD_LAST_EXIT=$?
	export -p > ${MCE_DIR}/env.sh
	pwd > ${MCE_DIR}/dir.txt
	exit $CODEBUILD_LAST_EXIT`, sce.mceDir, strings.Join(command, " "))
	scriptName := fmt.Sprintf("script-%d.sh", time.Now().UnixNano())
	scriptPath := filepath.Join(sce.mceDir, scriptName)
	if err := os.WriteFile(scriptPath, []byte(script), 00755); err != nil /* #nosec G306 */ {
		return err
	}

	cmd := exec.CommandContext(ctx, "/bin/sh", scriptPath) //#nosec G204
	cmd.Stdin = nil
	cmd.Dir = sce.WorkingDir
	cmd.Env = sce.Env

	log.Debug().Msgf("ExecuteCommand: path=%s args=%s dir=%s env=%#v script=%s", cmd.Path, cmd.Args, cmd.Dir, cmd.Env, script)

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

func setOutputs(mceDir string, stdout io.Writer) common.Executor {
	return func(context.Context) error {
		f, err := os.Open(filepath.Join(mceDir, "env.sh"))
		if err != nil {
			return nil
		}
		defer f.Close()
		scanner := bufio.NewScanner(f)
		pattern := regexp.MustCompile(`^export (.+)="(.+)"$`)
		for scanner.Scan() {
			line := scanner.Text()
			kv := pattern.FindStringSubmatch(line)
			if len(kv) == 3 {
				fmt.Fprintf(stdout, "::set-output name=%s::%s\n", kv[1], kv[2])
			}
		}
		return scanner.Err()
	}
}

func copyOut(ctx context.Context, mceDir string, workingDir string, filemap *FileMap) common.Executor {
	sourcePath := resolvePath(filemap.SourcePath, mceDir)
	targetPath := resolvePath(filemap.TargetPath, workingDir)
	return func(context.Context) error {
		sources, err := filepath.Glob(sourcePath)
		if err != nil {
			return err
		}
		log.Debug().Msgf("clearing cache %s", targetPath)
		if err := os.RemoveAll(targetPath); err != nil {
			return err
		}
		log.Debug().Msgf("copying %v (%s) to %s", sources, sourcePath, targetPath)
		for _, source := range sources {
			if err := copyDir(
				ctx,
				targetPath,
				source,
				false,
			); err != nil {
				return err
			}
		}
		return nil
	}
}

func symlink(mceDir string, workingDir string, filemap *FileMap) error {
	if resolvePath(filemap.SourcePath, workingDir) != resolvePath(".", workingDir) {
		sourcePath := resolvePath(filemap.SourcePath, workingDir)
		targetPath := resolvePath(filemap.TargetPath, mceDir)
		log.Debug().Msgf("symlink mount %s to %s", sourcePath, targetPath)
		if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
			return err
		}
		return os.Symlink(sourcePath, targetPath)
	}
	return nil
}

func interpolate(s string, vars map[string]string) string {
	r := regexp.MustCompile(`\${?([a-zA-Z0-9_\-.]+)}?`)
	symbols := regexp.MustCompile(`[${}]`)
	repl := func(match string) string {
		key := symbols.ReplaceAllString(match, "")
		if val, ok := vars[key]; ok {
			return val
		}
		return match
	}
	rtn := r.ReplaceAllStringFunc(s, repl)
	log.Debug().Msgf("interpolate %s -> %s", s, rtn)
	return rtn
}

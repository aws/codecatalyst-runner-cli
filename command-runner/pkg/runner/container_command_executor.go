package runner

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/aws/codecatalyst-runner-cli/command-runner/internal/containers/types"
	"github.com/aws/codecatalyst-runner-cli/command-runner/internal/fs"
	"github.com/aws/codecatalyst-runner-cli/command-runner/pkg/common"

	"github.com/opencontainers/selinux/go-selinux"
	"github.com/rs/zerolog/log"
)

type containerCommandExecutor struct {
	Container       types.Container
	ReuseContainers bool
	CloseExecutors  []common.Executor
	mceDir          string
	ctx             context.Context
}

type newContainerCommandExecutorParams struct {
	*EnvironmentConfiguration
	ID               string
	Image            string
	Entrypoint       Command
	ContainerService types.ContainerService
}

const containerSourceDir = "/codecatalyst/output/src"

func newContainerCommandExecutor(ctx context.Context, params *newContainerCommandExecutorParams) (commandExecutor, error) {
	containerName := fmt.Sprintf("codecatalyst-%s", regexp.MustCompile(`[^a-zA-Z0-9_.-]`).ReplaceAllString(params.ID, "_"))
	containerName = strings.ToLower(containerName)

	var imagePrep common.Executor
	var image string
	platform := ""

	if i, found := strings.CutPrefix(params.Image, "docker://"); found {
		// pull image from registry
		image = i
	} else {
		// local docker build
		dockerfilePath := params.Image
		if !filepath.IsAbs(dockerfilePath) {
			dockerfilePath = filepath.Join(params.WorkingDir, params.Image)
		}
		_, err := os.Stat(dockerfilePath)
		if err != nil {
			return nil, err
		}
		image = fmt.Sprintf("%s:%s", containerName, "latest")
		exists, err := params.ContainerService.ImageExistsLocally(ctx, image, platform)
		log.Ctx(ctx).Debug().Msgf("%s exists? %v", image, exists)
		if err != nil {
			log.Ctx(ctx).Error().Err(err).Msg("unable to check for local image")
		}
		if params.Reuse && exists {
			imagePrep = func(ctx context.Context) error { return nil }
		} else {
			imagePrep = params.ContainerService.BuildImage(types.BuildImageInput{
				ContextDir: filepath.Dir(dockerfilePath),
				Dockerfile: filepath.Base(dockerfilePath),
				ImageTag:   image,
				Platform:   platform,
			}).TraceRegion("image-build")
		}
	}

	env, containerDefaultDir, err := setupEnvironmentVariables(params.Env)
	if err != nil {
		return nil, err
	}

	mceDir, err := setupMceDir(env, containerDefaultDir)
	if err != nil {
		return nil, err
	}
	binds := []string{
		fmt.Sprintf("%s:%s", "/var/run/docker.sock", "/var/run/docker.sock"),
		fmt.Sprintf("%s:%s", mceDir, "/tmp/mce"),
	}

	for _, filemap := range params.FileMaps {
		srcPath := resolvePath(filemap.SourcePath, params.WorkingDir)
		targetPath := resolvePath(filemap.TargetPath, containerSourceDir)
		if filemap.Type == FileMapTypeBind {
			binds = append(binds,
				fmt.Sprintf(
					"%s:%s%s",
					srcPath,
					targetPath,
					bindModifiers(),
				),
			)
		}
	}

	log.Ctx(ctx).Debug().Msgf("Container binds: %#v", binds)
	log.Ctx(ctx).Debug().Msgf("Container env: %#v", env)

	actionContainer := params.ContainerService.NewContainer(types.NewContainerInput{
		Image:      image,
		Name:       containerName,
		Stdout:     params.Stdout,
		Stderr:     params.Stderr,
		Env:        env,
		WorkingDir: containerDefaultDir,
		Binds:      binds,
		Entrypoint: params.Entrypoint,
	})

	copyExecutors, closeExecutors, err := setupCopyAndCloseExecutors(actionContainer, params.WorkingDir, params.FileMaps)
	if err != nil {
		return nil, err
	}

	if imagePrep == nil {
		imagePrep = actionContainer.Pull(true).TraceRegion("image-pull")
	}

	if err := common.NewPipelineExecutor(
		imagePrep,
		actionContainer.Remove().IfBool(!params.Reuse),
		actionContainer.Create(nil, nil).TraceRegion("container-create"),
		common.NewPipelineExecutor(copyExecutors...).TraceRegion("container-copy"),
		actionContainer.Start(false).TraceRegion("container-start"),
	)(ctx); err != nil {
		return nil, fmt.Errorf("unable to create container executor: %w", err)
	}

	return &containerCommandExecutor{
		Container:       actionContainer,
		ReuseContainers: params.Reuse,
		CloseExecutors:  closeExecutors,
		mceDir:          mceDir,
		ctx:             ctx,
	}, nil
}

func (cce *containerCommandExecutor) Close(isError bool) error {
	var err error
	if !isError {
		err = common.NewPipelineExecutor(cce.CloseExecutors...).TraceRegion("close-executors")(cce.ctx)
	}
	if !cce.ReuseContainers {
		if err := cce.Container.Remove()(context.Background()); err != nil {
			log.Ctx(cce.ctx).Error().Err(err).Msg("error removing container")
		}
	}
	if err := os.RemoveAll(cce.mceDir); err != nil {
		log.Ctx(cce.ctx).Error().Err(err).Msg("error removing temp mce directory")
	}
	return err
}

func (cce *containerCommandExecutor) ExecuteCommand(ctx context.Context, command Command) error {
	script := fmt.Sprintf(`cd $(cat /tmp/mce/tmp/dir.txt)
set -a
. /tmp/mce/tmp/env.sh
while read line; do
	env "$line" > /dev/null
done < /tmp/mce/tmp/init.env
%s
CODEBUILD_LAST_EXIT=$?
export -p > /tmp/mce/tmp/env.sh
pwd > /tmp/mce/tmp/dir.txt
exit $CODEBUILD_LAST_EXIT`, strings.Join(command, " "))
	log.Ctx(ctx).Debug().Msgf("script: %s", script)
	scriptName := fmt.Sprintf("script-%d.sh", time.Now().UnixNano())
	if err := os.WriteFile(filepath.Join(cce.mceDir, "tmp", scriptName), []byte(script), 00755); err != nil /* #nosec G306 */ {
		return err
	}

	return cce.Container.Exec([]string{"/bin/sh", fmt.Sprintf("/tmp/mce/tmp/%s", scriptName)}, nil, "", "")(ctx)
}

func bindModifiers() string {
	var bindModifiers string
	if runtime.GOOS == "darwin" {
		bindModifiers = ":consistent"
	}
	if selinux.GetEnabled() {
		bindModifiers = ":z"
	}
	return bindModifiers
}

func resolvePath(path string, basePath string) string {
	p := path
	if !filepath.IsAbs(p) {
		absBasePath, err := filepath.Abs(basePath)
		if err != nil {
			log.Fatal().Err(err)
		}
		p = fmt.Sprintf("%s/%s", absBasePath, p)
	}
	return p
}

func clean(dir string) common.Executor {
	return func(ctx context.Context) error {
		if err := os.RemoveAll(dir); err != nil {
			return err
		}
		return os.MkdirAll(dir, 0755)
	}
}

func setupMceDir(env []string, containerDefaultDir string) (string, error) {
	mceDir, err := os.MkdirTemp(fs.TmpDir(), "mce")
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Join(mceDir, "tmp"), 0755); err != nil {
		return "", fmt.Errorf("unable to create tmp dir: %w", err)
	}
	if err := os.WriteFile(filepath.Join(mceDir, "tmp", "init.env"), []byte(strings.Join(env, "\n")), 00777); err != nil /* #nosec G306 */ {
		return "", err
	}
	if err := os.WriteFile(filepath.Join(mceDir, "tmp", "env.sh"), []byte(""), 00666); err != nil /* #nosec G306 */ {
		return "", err
	}
	if err := os.WriteFile(filepath.Join(mceDir, "tmp", "dir.txt"), []byte(containerDefaultDir), 00666); err != nil /* #nosec G306 */ {
		return "", err
	}
	envout := `. /tmp/mce/tmp/env.sh
env -0 | while IFS='=' read -r -d '' n v; do  printf "::set-output name=%s::%s\n" "$n" "$v"; done`
	if err := os.WriteFile(filepath.Join(mceDir, "tmp", "envout.sh"), []byte(envout), 00755); err != nil /* #nosec G306 */ {
		return "", err
	}
	return mceDir, nil
}

func setupCopyAndCloseExecutors(actionContainer types.Container, workingDir string, filemaps []*FileMap) ([]common.Executor, []common.Executor, error) {
	copyExecutors := []common.Executor{}
	closeExecutors := []common.Executor{
		actionContainer.Exec([]string{"/bin/sh", "/tmp/mce/tmp/envout.sh"}, nil, "", "/"),
	}
	for _, filemap := range filemaps {
		switch filemap.Type {
		case FileMapTypeBind:
			continue
		case FileMapTypeCopyOut:
			srcPath := resolvePath(filemap.SourcePath, containerSourceDir)
			closeExecutors = append(
				closeExecutors,
				actionContainer.Exec([]string{"mkdir", "-p", "/extract"}, nil, "", "/"),
				actionContainer.Exec([]string{"/bin/sh", "-c", fmt.Sprintf("cp -a %s /extract || echo 'nothing to cache' > /dev/null 2>&1", srcPath)}, nil, "", "/"),
			)
			if !strings.HasSuffix(srcPath, "/.") {
				closeExecutors = append(closeExecutors, clean(filemap.TargetPath))
			}
			closeExecutors = append(
				closeExecutors,
				actionContainer.CopyOut(filemap.TargetPath, "/extract/."),
				actionContainer.Exec([]string{"rm", "-rf", "/extract"}, nil, "", "/"),
			)

		case FileMapTypeCopyIn:
			copyExecutors = append(
				copyExecutors,
				actionContainer.CopyIn(
					resolvePath(filemap.TargetPath, containerSourceDir),
					resolvePath(filemap.SourcePath, workingDir),
					false,
				),
			)
		case FileMapTypeCopyInWithGitignore:
			copyExecutors = append(
				copyExecutors,
				actionContainer.CopyIn(
					resolvePath(filemap.TargetPath, containerSourceDir),
					resolvePath(filemap.SourcePath, workingDir),
					true,
				),
			)
		default:
			return nil, nil, fmt.Errorf("unknown filemap Type")
		}
	}
	return copyExecutors, closeExecutors, nil
}

func setupEnvironmentVariables(env map[string]string) ([]string, string, error) {
	var containerDefaultDir string
	envVars := make([]string, 0)
	for k, v := range env {
		if strings.HasPrefix(k, "CATALYST_SOURCE_DIR_") {
			v = resolvePath(v, containerSourceDir)
			if k == "CATALYST_SOURCE_DIR_WorkflowSource" || containerDefaultDir == "" {
				containerDefaultDir = v
			}
		}
		envVars = append(envVars, fmt.Sprintf("%s=%s", k, v))
	}

	if containerDefaultDir == "" {
		return nil, "", fmt.Errorf("input source or artifact is required")
	}
	envVars = append(envVars, fmt.Sprintf("CATALYST_DEFAULT_DIR=%s", containerDefaultDir))
	return envVars, containerDefaultDir, nil
}

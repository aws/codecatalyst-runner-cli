package finch

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"

	"github.com/aws/codecatalyst-runner-cli/command-runner/internal/containers/types"
	"github.com/aws/codecatalyst-runner-cli/command-runner/internal/fs"
	"github.com/aws/codecatalyst-runner-cli/command-runner/pkg/common"

	"github.com/go-git/go-billy/v5/helper/polyfill"
	"github.com/go-git/go-billy/v5/osfs"
	"github.com/go-git/go-git/v5/plumbing/format/gitignore"

	"github.com/rs/zerolog/log"
)

func (cr *finchContainer) Create(capAdd []string, capDrop []string) common.Executor {
	return common.NewPipelineExecutor(
		cr.connect(),
		cr.find(),
		cr.create(capAdd, capDrop),
	).IfNot(common.Dryrun)
}

func (cr *finchContainer) Start(attach bool) common.Executor {
	return common.
		NewInfoExecutor("🐦 finch run image=%s", cr.input.Image).
		Then(
			common.NewPipelineExecutor(
				cr.connect(),
				cr.find(),
				cr.start(attach),
				cr.wait().IfBool(attach),
				cr.tryReadUID(),
				cr.tryReadGID(),
			).IfNot(common.Dryrun),
		)
}

func (cr *finchContainer) start(attach bool) common.Executor {
	return func(ctx context.Context) error {
		log.Ctx(ctx).Printf("Starting container: %v", cr.id)

		args := []string{"start", cr.id}
		if attach {
			args = append(args, "--attach")
		}
		if rout, rerr, err := cr.f.RunWithoutStdio(ctx, args...); err != nil {
			return fmt.Errorf("failed to start container: %w\n%s\n%s", err, rout, rerr)
		}
		log.Ctx(ctx).Printf("Started container: %v", cr.id)

		return nil
	}
}

func (cr *finchContainer) wait() common.Executor {
	return func(ctx context.Context) error {
		if rout, rerr, err := cr.f.RunWithoutStdio(ctx, "wait", cr.id); err != nil {
			return fmt.Errorf("failed to wait container: %w\n%s\n%s", err, rout, rerr)
		} else {
			statusCode := string(rout)
			if statusCode != "0" {
				return fmt.Errorf("exit with `FAILURE`: %v", statusCode)
			}
		}
		return nil
	}
}

func (cr *finchContainer) tryReadID(opt string, cbk func(id int)) common.Executor {
	return func(ctx context.Context) error {
		// TODO: implement
		return nil
	}
}

func (cr *finchContainer) tryReadUID() common.Executor {
	return cr.tryReadID("-u", func(id int) { cr.uid = id })
}

func (cr *finchContainer) tryReadGID() common.Executor {
	return cr.tryReadID("-g", func(id int) { cr.gid = id })
}

func (cr *finchContainer) Pull(forcePull bool) common.Executor {
	return common.
		NewInfoExecutor("🐦 finch pull image=%s", cr.input.Image).
		Then(
			newPullExecutor(newPullExecutorInput{
				Image:     cr.input.Image,
				ForcePull: forcePull,
				Platform:  cr.input.Platform,
				Username:  cr.input.Username,
				Password:  cr.input.Password,
			}),
		)
}

func (cr *finchContainer) CopyIn(containerPath string, hostPath string, useGitIgnore bool) common.Executor {
	return common.NewPipelineExecutor(
		common.NewDebugExecutor("🐦 finch copyIn hostPath=%s containerPath=%s", hostPath, containerPath),
		cr.connect(),
		cr.find(),
		cr.copyIn(containerPath, hostPath, useGitIgnore),
	).IfNot(common.Dryrun)
}

func (cr *finchContainer) CopyOut(hostPath string, containerPath string) common.Executor {
	return common.NewPipelineExecutor(
		common.NewDebugExecutor("🐦 finch copyOut hostPath=%s containerPath=%s", hostPath, containerPath),
		cr.connect(),
		cr.find(),
		cr.copyOut(hostPath, containerPath),
	).IfNot(common.Dryrun)
}

func (cr *finchContainer) Exec(command []string, env map[string]string, user, workdir string) common.Executor {
	return common.NewPipelineExecutor(
		common.NewDebugExecutor("🐦 finch exec cmd=[%s] user=%s workdir=%s", strings.Join(command, " "), user, workdir),
		cr.connect(),
		cr.find(),
		cr.exec(command, env, user, workdir),
	).IfNot(common.Dryrun)
}

func (cr *finchContainer) Remove() common.Executor {
	return common.NewPipelineExecutor(
		cr.connect(),
		cr.find(),
	).Finally(
		cr.remove(),
	).IfNot(common.Dryrun)
}

func (cr *finchContainer) connect() common.Executor {
	return func(ctx context.Context) error {
		if cr.f != nil {
			return nil
		}
		f, err := newFinch(finchInstallDir)
		if err != nil {
			return err
		}
		cr.f = f
		return nil
	}
}

type finchContainerSpec struct {
	ID    string      `json:"ID"`
	Names interface{} `json:"Names"`
}

func (cr *finchContainer) find() common.Executor {
	return func(ctx context.Context) error {
		if cr.id != "" {
			return nil
		}
		rout, rerr, err := cr.f.RunWithoutStdio(ctx, "container", "ls", "--all", "--format", "{{json .}}")
		if err != nil {
			return fmt.Errorf("failed to list containers: %w\n%s", err, rerr)
		}

		scanner := bufio.NewScanner(bytes.NewReader(rout))
		for scanner.Scan() {
			cs := new(finchContainerSpec)
			if err := json.Unmarshal(scanner.Bytes(), cs); err != nil {
				return fmt.Errorf("failed unmarshalling container spec: %w", err)
			}
			names := make([]string, 0)
			switch typedNames := cs.Names.(type) {
			case []interface{}:
				for _, inf := range typedNames {
					names = append(names, inf.(string))
				}
			case []string:
				names = typedNames
			case string:
				names = []string{typedNames}
			default:
				return fmt.Errorf("invalid names type: %T", cs.Names)
			}
			log.Ctx(ctx).Debug().Msgf("got back container %+v with names %v", cs, names)
			if slices.Contains(names, cr.input.Name) {
				cr.id = cs.ID
				return nil
			}
		}

		cr.id = ""
		return nil
	}
}

func (cr *finchContainer) create(capAdd []string, capDrop []string) common.Executor {
	return func(ctx context.Context) error {
		if cr.id != "" {
			return nil
		}
		input := cr.input

		flags := []string{"--tty", "--workdir", input.WorkingDir, "--name", input.Name}
		if input.Privileged {
			flags = append(flags, "--privileged")
		}
		for src, dst := range input.Mounts {
			flags = append(flags, "--mount", fmt.Sprintf("type=volume,src=%s,dst=%s", src, dst))
		}
		for _, bind := range input.Binds {
			flags = append(flags, "--volume", bind)
		}
		if len(capAdd) != 0 {
			flags = append(flags, "--cap-add")
			flags = append(flags, capAdd...)
		}
		if len(capDrop) != 0 {
			flags = append(flags, "--cap-drop")
			flags = append(flags, capDrop...)
		}
		for _, e := range input.Env {
			flags = append(flags, "--env", e)
		}
		if len(input.Entrypoint) != 0 {
			flags = append(flags, "--entrypoint")
			flags = append(flags, input.Entrypoint...)
		}

		args := []string{"create"}
		args = append(args, flags...)
		args = append(args, input.Image)
		if len(input.Cmd) != 0 {
			args = append(args, input.Cmd...)
		}
		rout, rerr, err := cr.f.RunWithoutStdio(ctx, args...)
		if err != nil {
			return fmt.Errorf("failed to create container: '%w'\n%s\n%s", err, rout, rerr)
		}

		id := strings.TrimRight(string(rout), "\r\n")

		log.Ctx(ctx).Printf("Created container name=%s id=%v from image %v (platform: %s)", input.Name, id, input.Image, input.Platform)
		log.Ctx(ctx).Printf("ENV ==> %v", input.Env)

		cr.id = id
		return nil
	}
}
func (cr *finchContainer) remove() common.Executor {
	return func(ctx context.Context) error {
		if cr.id == "" {
			return nil
		}
		log.Ctx(ctx).Debug().Msgf("🐦 finch rm %s", cr.id)

		_, lerr, err := cr.f.RunWithoutStdio(ctx, "rm", "--force", "--volumes", cr.id)
		if err != nil {
			return fmt.Errorf("failed to list containers: %w\n%s", err, lerr)
		}
		if err != nil {
			log.Ctx(ctx).Err(fmt.Errorf("failed to remove container: %w", err))
		}

		log.Ctx(ctx).Printf("Removed container: %v", cr.id)
		cr.id = ""
		return nil
	}
}

func (cr *finchContainer) Close() common.Executor {
	return func(ctx context.Context) error {
		return nil
	}
}

type finchContainer struct {
	id    string
	input types.NewContainerInput
	f     *finch
	uid   int
	gid   int
}

func (cr *finchContainer) copyIn(containerPath string, hostPath string, useGitIgnore bool) common.Executor {
	return func(ctx context.Context) error {
		log.Ctx(ctx).Debug().Msgf("Writing %s from %s", containerPath, hostPath)
		if f, err := os.Stat(hostPath); err != nil {
			return fmt.Errorf("unable to copyIn from hostPath=%s: %w", hostPath, err)
		} else if filepath.Ext(hostPath) == ".tar" {
			// TODO: implement
			return fmt.Errorf("copyIn for tar file is not implemented")
		} else if f.IsDir() {
			if useGitIgnore {
				tempDir, err := os.MkdirTemp(fs.TmpDir(), "finch-copyin")
				if err != nil {
					return fmt.Errorf("failed to create temp dir: %w", err)
				}
				defer os.RemoveAll(tempDir)

				err = copyDir(ctx, tempDir, hostPath, useGitIgnore)
				if err != nil {
					return fmt.Errorf("failed to copyDir: %w", err)
				}
				hostPath = fmt.Sprintf("%s/.", tempDir)
			}
			if _, rerr, err := cr.f.RunWithoutStdio(ctx, "cp", hostPath, fmt.Sprintf("%s:%s", cr.id, containerPath)); err != nil {
				return fmt.Errorf("failed to copy content to container: %w\n%s", err, rerr)
			}
			return nil
		} else {
			return fmt.Errorf("unsupported srcPath=%s", hostPath)
		}
	}
}

func (cr *finchContainer) copyOut(hostPath string, containerPath string) common.Executor {
	return func(ctx context.Context) error {
		if err := os.MkdirAll(filepath.Dir(hostPath), 0755); err != nil {
			return fmt.Errorf("failed to create hostPath=%s: %w", hostPath, err)
		}
		log.Ctx(ctx).Debug().Msgf("Writing %s from %s", hostPath, containerPath)
		if _, rerr, err := cr.f.RunWithoutStdio(ctx, "cp", fmt.Sprintf("%s:%s", cr.id, containerPath), hostPath); err != nil {
			return fmt.Errorf("failed to copy content from container: %w\n%s", err, rerr)
		}
		return nil
	}
}

func (cr *finchContainer) exec(cmd []string, env map[string]string, user, workdir string) common.Executor {
	return func(ctx context.Context) error {
		// Fix slashes when running on Windows
		if runtime.GOOS == "windows" {
			var newCmd []string
			for _, v := range cmd {
				newCmd = append(newCmd, strings.ReplaceAll(v, `\`, `/`))
			}
			cmd = newCmd
		}

		log.Ctx(ctx).Printf("Exec command '%s'", cmd)

		var wd string
		if workdir != "" {
			if strings.HasPrefix(workdir, "/") {
				wd = workdir
			} else {
				wd = fmt.Sprintf("%s/%s", cr.input.WorkingDir, workdir)
			}
		} else {
			wd = cr.input.WorkingDir
		}
		log.Ctx(ctx).Printf("Working directory '%s'", wd)

		flags := []string{"--workdir", wd}
		if user != "" {
			flags = append(flags, "--user", user)
		}
		for k, v := range env {
			flags = append(flags, "--env", fmt.Sprintf("%s=%s", k, v))
		}

		args := []string{"exec"}
		args = append(args, flags...)
		args = append(args, cr.id)
		args = append(args, cmd...)
		err := cr.f.RunWithStdio(ctx, nil, cr.input.Stdout, cr.input.Stderr, args...)
		if err != nil {
			return fmt.Errorf("exec failed: %w", err)
		}
		return err
	}
}

func copyDir(ctx context.Context, destdir string, sourcedir string, useGitIgnore bool) error {
	if sourcedir == destdir {
		return fmt.Errorf("unable to copyDir when sourcedir==destdir: %s", destdir)
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

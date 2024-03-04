package docker

import (
	"archive/tar"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"

	"github.com/aws/codecatalyst-runner-cli/command-runner/internal/containers/shared"
	ctypes "github.com/aws/codecatalyst-runner-cli/command-runner/internal/containers/types"
	"github.com/aws/codecatalyst-runner-cli/command-runner/pkg/common"

	"github.com/rs/zerolog/log"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
	specs "github.com/opencontainers/image-spec/specs-go/v1"
)

func (cr *dockerContainer) Create(capAdd []string, capDrop []string) common.Executor {
	return common.
		NewDebugExecutor("%sdocker create image=%s platform=%s entrypoint=%+q cmd=%+q", logPrefix, cr.input.Image, cr.input.Platform, cr.input.Entrypoint, cr.input.Cmd).
		Then(
			common.NewPipelineExecutor(
				cr.connect(),
				cr.find(),
				cr.create(capAdd, capDrop),
			).IfNot(common.Dryrun),
		)
}

func (cr *dockerContainer) Start(attach bool) common.Executor {
	return common.
		NewInfoExecutor("%sdocker run image=%s", logPrefix, cr.input.Image).
		Then(
			common.NewPipelineExecutor(
				cr.connect(),
				cr.find(),
				cr.attach().IfBool(attach),
				cr.start(),
				cr.wait().IfBool(attach),
				cr.tryReadUID(),
				cr.tryReadGID(),
			).IfNot(common.Dryrun),
		)
}

func (cr *dockerContainer) Pull(forcePull bool) common.Executor {
	return common.
		NewInfoExecutor("%sdocker pull image=%s", logPrefix, cr.input.Image).
		Then(
			newDockerPullExecutor(newDockerPullExecutorInput{
				Image:     cr.input.Image,
				ForcePull: forcePull,
				Platform:  cr.input.Platform,
				Username:  cr.input.Username,
				Password:  cr.input.Password,
			}),
		)
}

func (cr *dockerContainer) CopyIn(containerPath string, hostPath string, useGitIgnore bool) common.Executor {
	return common.NewPipelineExecutor(
		common.NewDebugExecutor("%sdocker cp hostPath=%s containerPath=%s", logPrefix, hostPath, containerPath),
		cr.copyIn(containerPath, hostPath, useGitIgnore),
	).IfNot(common.Dryrun)
}

func (cr *dockerContainer) CopyOut(hostPath string, containerPath string) common.Executor {
	return common.NewPipelineExecutor(
		common.NewDebugExecutor("%sdocker cp hostPath=%s containerPath=%s", logPrefix, hostPath, containerPath),
		cr.copyOut(hostPath, containerPath),
	).IfNot(common.Dryrun)
}

func (cr *dockerContainer) Exec(command []string, env map[string]string, user, workdir string) common.Executor {
	return common.NewPipelineExecutor(
		common.NewDebugExecutor("%sdocker exec cmd=[%s] user=%s workdir=%s", logPrefix, strings.Join(command, " "), user, workdir),
		cr.connect(),
		cr.find(),
		cr.exec(command, env, user, workdir),
	).IfNot(common.Dryrun)
}

func (cr *dockerContainer) Remove() common.Executor {
	return common.NewPipelineExecutor(
		cr.connect(),
		cr.find(),
	).Finally(
		cr.remove(),
	).IfNot(common.Dryrun)
}

type dockerContainer struct {
	cli   client.APIClient
	id    string
	input ctypes.NewContainerInput
	UID   int
	GID   int
}

func (cr *dockerContainer) Close() common.Executor {
	return func(ctx context.Context) error {
		if cr.cli != nil {
			err := cr.cli.Close()
			cr.cli = nil
			if err != nil {
				return fmt.Errorf("failed to close client: %w", err)
			}
		}
		return nil
	}
}

func (cr *dockerContainer) find() common.Executor {
	return func(ctx context.Context) error {
		if cr.id != "" {
			return nil
		}
		containers, err := cr.cli.ContainerList(ctx, container.ListOptions{
			All: true,
		})
		if err != nil {
			return fmt.Errorf("failed to list containers: %w", err)
		}

		for _, c := range containers {
			for _, name := range c.Names {
				if name[1:] == cr.input.Name {
					cr.id = c.ID
					return nil
				}
			}
		}

		cr.id = ""
		return nil
	}
}

func (cr *dockerContainer) remove() common.Executor {
	return func(ctx context.Context) error {
		if cr.id == "" {
			return nil
		}

		err := cr.cli.ContainerRemove(ctx, cr.id, container.RemoveOptions{
			RemoveVolumes: true,
			Force:         true,
		})
		if err != nil {
			log.Ctx(ctx).Err(fmt.Errorf("failed to remove container: %w", err))
		}

		log.Ctx(ctx).Printf("Removed container: %v", cr.id)
		cr.id = ""
		return nil
	}
}

func (cr *dockerContainer) create(capAdd []string, capDrop []string) common.Executor {
	return func(ctx context.Context) error {
		if cr.id != "" {
			return nil
		}
		input := cr.input

		config := &container.Config{
			Image:      input.Image,
			WorkingDir: input.WorkingDir,
			Env:        input.Env,
			Tty:        true,
		}
		log.Ctx(ctx).Printf("Common container.Config ==> %+v", config)

		if len(input.Cmd) != 0 {
			config.Cmd = input.Cmd
		}

		if len(input.Entrypoint) != 0 {
			config.Entrypoint = input.Entrypoint
		}

		mounts := make([]mount.Mount, 0)
		for mountSource, mountTarget := range input.Mounts {
			mounts = append(mounts, mount.Mount{
				Type:        mount.TypeVolume,
				Source:      mountSource,
				Target:      mountTarget,
				Consistency: mount.ConsistencyDefault,
			})
		}

		var platSpecs *specs.Platform
		if cr.input.Platform != "" {
			desiredPlatform := strings.SplitN(cr.input.Platform, `/`, 2)

			if len(desiredPlatform) != 2 {
				return fmt.Errorf("incorrect container platform option '%s'", cr.input.Platform)
			}

			platSpecs = &specs.Platform{
				Architecture: desiredPlatform[1],
				OS:           desiredPlatform[0],
			}
		}

		hostConfig := &container.HostConfig{
			CapAdd:      capAdd,
			CapDrop:     capDrop,
			Binds:       input.Binds,
			Mounts:      mounts,
			NetworkMode: container.NetworkMode(input.NetworkMode),
			Privileged:  input.Privileged,
			UsernsMode:  container.UsernsMode(input.UsernsMode),
		}
		log.Ctx(ctx).Printf("Common container.HostConfig ==> %+v", hostConfig)

		resp, err := cr.cli.ContainerCreate(ctx, config, hostConfig, nil, platSpecs, input.Name)
		if err != nil {
			return fmt.Errorf("failed to create container: '%w'", err)
		}

		log.Ctx(ctx).Printf("Created container name=%s id=%v from image %v (platform: %s)", input.Name, resp.ID, input.Image, input.Platform)
		log.Ctx(ctx).Printf("ENV ==> %v", input.Env)

		cr.id = resp.ID
		return nil
	}
}

func (cr *dockerContainer) exec(cmd []string, env map[string]string, user, workdir string) common.Executor {
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
		envList := make([]string, 0)
		for k, v := range env {
			envList = append(envList, fmt.Sprintf("%s=%s", k, v))
		}

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

		idResp, err := cr.cli.ContainerExecCreate(ctx, cr.id, types.ExecConfig{
			User:         user,
			Cmd:          cmd,
			WorkingDir:   wd,
			Env:          envList,
			Tty:          true,
			AttachStderr: true,
			AttachStdout: true,
		})
		if err != nil {
			return fmt.Errorf("failed to create exec: %w", err)
		}

		resp, err := cr.cli.ContainerExecAttach(ctx, idResp.ID, types.ExecStartCheck{
			Tty: true,
		})
		if err != nil {
			return fmt.Errorf("failed to attach to exec: %w", err)
		}
		defer resp.Close()

		err = cr.waitForCommand(ctx, resp, idResp, user, workdir)
		if err != nil {
			return err
		}

		inspectResp, err := cr.cli.ContainerExecInspect(ctx, idResp.ID)
		if err != nil {
			return fmt.Errorf("failed to inspect exec: %w", err)
		}
		log.Ctx(ctx).Debug().Msgf("Got back exec inspect=%#v", inspectResp)

		switch inspectResp.ExitCode {
		case 0:
			return nil
		case 127:
			return fmt.Errorf("exitcode '%d': command not found", inspectResp.ExitCode)
		default:
			return fmt.Errorf("exitcode '%d': failure", inspectResp.ExitCode)
		}
	}
}

func (cr *dockerContainer) tryReadID(opt string, cbk func(id int)) common.Executor {
	return func(ctx context.Context) error {
		idResp, err := cr.cli.ContainerExecCreate(ctx, cr.id, types.ExecConfig{
			Cmd:          []string{"id", opt},
			AttachStdout: true,
			AttachStderr: true,
		})
		if err != nil {
			log.Ctx(ctx).Debug().Err(err).Msgf("tryReadID - Unable to create exec for container=%s", cr.id)
			inspectResp, err := cr.cli.ContainerInspect(ctx, cr.id)
			if err != nil {
				log.Ctx(ctx).Debug().Err(err).Msgf("tryReadID - Unable to inspect container=%s", cr.id)
			} else {
				log.Ctx(ctx).Debug().Msgf("state=%#v", inspectResp.State)
				logResp, err := cr.cli.ContainerLogs(ctx, cr.id, container.LogsOptions{
					ShowStdout: true,
					ShowStderr: true,
				})
				if err != nil {
					log.Ctx(ctx).Debug().Err(err).Msgf("tryReadID - Unable to get logs for container=%s", cr.id)
				}
				if log.Ctx(ctx).Debug().Enabled() {
					_, _ = io.Copy(os.Stdout, logResp)
				}
			}
			return nil
		}

		resp, err := cr.cli.ContainerExecAttach(ctx, idResp.ID, types.ExecStartCheck{})
		if err != nil {
			log.Ctx(ctx).Warn().Err(err).Msgf("tryReadID - Unable to attach exec for container=%s", cr.id)
			return nil
		}
		defer resp.Close()

		sid, err := resp.Reader.ReadString('\n')
		if err != nil {
			return nil
		}
		log.Ctx(ctx).Debug().Msgf("Reading id with opt=%s and got back: %s", opt, sid)
		exp := regexp.MustCompile(`\d+\n`)
		found := exp.FindString(sid)
		if len(found) == 0 {
			log.Ctx(ctx).Warn().Msgf("Unable to read id with opt=%s - got back: %s", opt, sid)
			return nil
		}
		id, err := strconv.ParseInt(found[:len(found)-1], 10, 32)
		if err != nil {
			return nil
		}
		cbk(int(id))

		return nil
	}
}

func (cr *dockerContainer) tryReadUID() common.Executor {
	return cr.tryReadID("-u", func(id int) { cr.UID = id })
}

func (cr *dockerContainer) tryReadGID() common.Executor {
	return cr.tryReadID("-g", func(id int) { cr.GID = id })
}

func (cr *dockerContainer) waitForCommand(ctx context.Context, resp types.HijackedResponse, _ types.IDResponse, _ string, _ string) error {
	cmdResponse := make(chan error)

	go func() {
		var outWriter io.Writer
		outWriter = cr.input.Stdout
		if outWriter == nil {
			outWriter = os.Stdout
		}
		_, err := io.Copy(outWriter, resp.Reader)
		cmdResponse <- err
	}()

	select {
	case <-ctx.Done():
		// send ctrl + c
		_, err := resp.Conn.Write([]byte{3})
		if err != nil {
			log.Ctx(ctx).Warn().Msgf("Failed to send CTRL+C: %+s", err)
		}

		// we return the context canceled error to prevent other steps
		// from executing
		return ctx.Err()
	case err := <-cmdResponse:
		if err != nil {
			log.Ctx(ctx).Err(err)
		}

		return nil
	}
}

func (cr *dockerContainer) copyIn(containerPath string, hostPath string, useGitIgnore bool) common.Executor {
	return func(ctx context.Context) error {
		log.Ctx(ctx).Debug().Msgf("Writing %s from %s", containerPath, hostPath)
		var tarFile *os.File
		if fs, err := os.Stat(hostPath); err != nil {
			return fmt.Errorf("unable to copyDir from srcPath=%s: %w", hostPath, err)
		} else if filepath.Ext(hostPath) == ".tar" {
			tarFile, err = os.Open(hostPath)
			if err != nil {
				return err
			}
			defer tarFile.Close()
		} else if fs.IsDir() {
			tarFile, err = shared.TarDirectory(ctx, hostPath, containerPath[1:], useGitIgnore, cr.UID, cr.GID)
			if tarFile != nil {
				defer func(tarFile *os.File) {
					if err := tarFile.Close(); err != nil && !errors.Is(err, os.ErrClosed) {
						log.Ctx(ctx).Err(err)
					}
					if err := os.Remove(tarFile.Name()); err != nil {
						log.Ctx(ctx).Err(err)
					}
				}(tarFile)
			}
			if err != nil {
				return err
			}
		} else {
			return fmt.Errorf("unsupported hostPath=%s", hostPath)
		}

		log.Ctx(ctx).Printf("Extracting content from '%s' to '%s'", tarFile.Name(), containerPath)
		if err := cr.cli.CopyToContainer(ctx, cr.id, "/", tarFile, types.CopyToContainerOptions{}); err != nil {
			return fmt.Errorf("failed to copy content to container: %w", err)
		}
		return nil
	}
}
func (cr *dockerContainer) copyOut(hostPath string, containerPath string) common.Executor {
	return func(ctx context.Context) error {
		log.Ctx(ctx).Printf("Extracting content from '%s' to '%s'", containerPath, hostPath)
		if out, _, err := cr.cli.CopyFromContainer(ctx, cr.id, containerPath); err != nil {
			return fmt.Errorf("failed to copy content from container: %w", err)
		} else {
			tr := tar.NewReader(out)
			defer out.Close()
			for {
				hdr, err := tr.Next()
				if err == io.EOF {
					break
				}
				if err != nil {
					return fmt.Errorf("error processing archive: %w", err)
				}
				hostPath := filepath.Join(hostPath, hdr.Name) // #nosec G305 - mitigated by following check
				if !strings.HasPrefix(hostPath, filepath.Clean(hostPath)) {
					return fmt.Errorf("content path is tainted: %s", hostPath)
				}
				switch hdr.Typeflag {
				case tar.TypeDir:
					if err := os.MkdirAll(hostPath, 0755); err != nil {
						return err
					}
				case tar.TypeReg:
					if err := os.MkdirAll(filepath.Dir(hostPath), 0755); err != nil {
						return err
					}
					if f, err := os.Create(hostPath); err != nil {
						return err
					} else {
						for {
							_, err := io.CopyN(f, tr, 1024)
							if err != nil {
								if err == io.EOF {
									break
								}
								return fmt.Errorf("unable to copy tr file to output: %w", err)
							}
						}
					}
				}
			}
		}
		return nil
	}
}

func (cr *dockerContainer) attach() common.Executor {
	return func(ctx context.Context) error {
		out, err := cr.cli.ContainerAttach(ctx, cr.id, container.AttachOptions{
			Stream: true,
			Stdout: true,
			Stderr: true,
		})
		if err != nil {
			return fmt.Errorf("failed to attach to container: %w", err)
		}

		var outWriter io.Writer
		outWriter = cr.input.Stdout
		if outWriter == nil {
			outWriter = os.Stdout
		}
		go func() {
			_, err = io.Copy(outWriter, out.Reader)
			if err != nil {
				log.Ctx(ctx).Err(err)
			}
		}()
		return nil
	}
}

func (cr *dockerContainer) start() common.Executor {
	return func(ctx context.Context) error {
		log.Ctx(ctx).Printf("Starting container: %v", cr.id)

		if err := cr.cli.ContainerStart(ctx, cr.id, container.StartOptions{}); err != nil {
			return fmt.Errorf("failed to start container: %w", err)
		}

		log.Ctx(ctx).Printf("Started container: %v", cr.id)
		return nil
	}
}

func (cr *dockerContainer) wait() common.Executor {
	return func(ctx context.Context) error {
		statusCh, errCh := cr.cli.ContainerWait(ctx, cr.id, container.WaitConditionNotRunning)
		var statusCode int64
		select {
		case err := <-errCh:
			if err != nil {
				return fmt.Errorf("failed to wait for container: %w", err)
			}
		case status := <-statusCh:
			statusCode = status.StatusCode
		}

		log.Ctx(ctx).Printf("Return status: %v", statusCode)

		if statusCode == 0 {
			return nil
		}

		return fmt.Errorf("exit with `FAILURE`: %v", statusCode)
	}
}

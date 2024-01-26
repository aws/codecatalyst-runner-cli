package finch

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"

	"github.com/aws/codecatalyst-runner-cli/command-runner/internal/containers/types"
	"github.com/aws/codecatalyst-runner-cli/command-runner/pkg/common"

	"github.com/rs/zerolog/log"
)

type ServiceProvider struct {
	available *bool
	mutex     sync.Mutex
}

func (f *ServiceProvider) NewContainerService() types.ContainerService {
	return &finchContainerService{}
}

type finchContainerService struct{}

func (f *ServiceProvider) Available(ctx context.Context) bool {
	f.mutex.Lock()
	defer f.mutex.Unlock()
	if f.available != nil {
		return *f.available
	}
	var avail bool
	if os.Getenv("NOFINCH") != "" {
		avail = false
	} else if f, err := newFinch(finchInstallDir); err != nil {
		log.Ctx(ctx).Debug().Err(err).Msg("finch is not installed")
		avail = false
	} else if rout, rerr, err := f.RunWithoutStdio(ctx, "container", "ls"); err != nil {
		log.Ctx(ctx).Debug().Err(err).Msgf("finch is unavailable: %s\n%s", rout, rerr)
		avail = false
	} else {
		avail = true
	}
	f.available = &avail
	return *f.available
}

func (fcs *finchContainerService) NewContainer(input types.NewContainerInput) types.Container {
	cr := new(finchContainer)
	cr.input = input
	return cr
}

// ImageExistsLocally returns a boolean indicating if an image with the
// requested name, tag and architecture exists in the local docker image store
func (fcs *finchContainerService) ImageExistsLocally(ctx context.Context, imageName string, platform string) (bool, error) {
	return imageExistsLocally(ctx, imageName, platform)
}

const finchInstallDir = "/Applications/Finch"

type finch struct {
	installDir string
}

func newFinch(installDir string) (*finch, error) {
	if stat, err := os.Stat(installDir); err != nil {
		return nil, err
	} else if !stat.IsDir() {
		return nil, fmt.Errorf("invalid finch installation directory %s", installDir)
	}

	return &finch{
		installDir: installDir,
	}, nil
}

func (f *finch) RunWithoutStdio(ctx context.Context, args ...string) ([]byte, []byte, error) {
	return f.RunWithStdin(ctx, nil, args...)
}

func (f *finch) RunWithStdin(ctx context.Context, in io.Reader, args ...string) ([]byte, []byte, error) {
	var bout bytes.Buffer
	var berr bytes.Buffer
	err := f.RunWithStdio(ctx, in, &bout, &berr, args...)
	return bout.Bytes(), berr.Bytes(), err
}

func (f *finch) RunWithStdio(ctx context.Context, stdin io.Reader, stdout io.Writer, stderr io.Writer, args ...string) error {
	finchCmd := fmt.Sprintf("%s/bin/finch", f.installDir)

	args = append([]string{finchCmd}, args...)
	cmd := exec.CommandContext(ctx, finchCmd) //#nosec G204
	cmd.Path = finchCmd
	cmd.Args = args
	cmd.Stdin = stdin

	cmdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	cmderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}

	log.Ctx(ctx).Debug().Msgf("🐦 %s", strings.Join(args, " "))
	if common.Dryrun(ctx) {
		log.Ctx(ctx).Debug().Msgf("exit for dryrun")
		return nil
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("unable to start command: %w", err)
	}
	if stdout != nil {
		go streamPipe(stdout, cmdout)
	}
	if stderr != nil {
		go streamPipe(stderr, cmderr)
	}
	err = cmd.Wait()
	if err != nil {
		return fmt.Errorf("finch command failed: %w", err)
	}
	return nil
}

func streamPipe(dst io.Writer, src io.ReadCloser) {
	reader := bufio.NewReader(src)
	_, _ = io.Copy(dst, reader)
}

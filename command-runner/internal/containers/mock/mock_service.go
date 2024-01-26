package mock

import (
	"context"

	"github.com/aws/codecatalyst-runner-cli/command-runner/internal/containers/types"
	"github.com/aws/codecatalyst-runner-cli/command-runner/pkg/common"

	"github.com/stretchr/testify/mock"
)

type MockContainerService struct {
	mock.Mock
}

func (mcs *MockContainerService) NewContainer(input types.NewContainerInput) types.Container {
	args := mcs.Called(input)
	return args.Get(0).(types.Container)
}

// ImageExistsLocally returns a boolean indicating if an image with the
// requested name, tag and architecture exists in the local docker image store
func (mcs *MockContainerService) ImageExistsLocally(ctx context.Context, imageName string, platform string) (bool, error) {
	args := mcs.Called(ctx, imageName, platform)
	return args.Bool(0), args.Error(1)
}

func (mcs *MockContainerService) BuildImage(input types.BuildImageInput) common.Executor {
	args := mcs.Called(input)
	return func(ctx context.Context) error {
		return args.Error(0)
	}
}

type MockContainer struct {
	mock.Mock
}

func (mc *MockContainer) Create(capAdd []string, capDrop []string) common.Executor {
	args := mc.Called(capAdd, capDrop)
	return func(ctx context.Context) error {
		return args.Error(0)
	}
}
func (mc *MockContainer) CopyIn(containerPath string, hostPath string, useGitIgnore bool) common.Executor {
	args := mc.Called(containerPath, hostPath, useGitIgnore)
	return func(ctx context.Context) error {
		return args.Error(0)
	}
}
func (mc *MockContainer) CopyOut(hostPath string, containerPath string) common.Executor {
	args := mc.Called(hostPath, containerPath)
	return func(ctx context.Context) error {
		return args.Error(0)
	}
}
func (mc *MockContainer) Pull(forcePull bool) common.Executor {
	args := mc.Called(forcePull)
	return func(ctx context.Context) error {
		return args.Error(0)
	}
}
func (mc *MockContainer) Start(attach bool) common.Executor {
	args := mc.Called(attach)
	return func(ctx context.Context) error {
		return args.Error(0)
	}
}
func (mc *MockContainer) Exec(command []string, env map[string]string, user, workdir string) common.Executor {
	args := mc.Called(command, env, user, workdir)
	return func(ctx context.Context) error {
		return args.Error(0)
	}
}
func (mc *MockContainer) Remove() common.Executor {
	args := mc.Called()
	return func(ctx context.Context) error {
		return args.Error(0)
	}
}
func (mc *MockContainer) Close() common.Executor {
	args := mc.Called()
	return func(ctx context.Context) error {
		return args.Error(0)
	}
}

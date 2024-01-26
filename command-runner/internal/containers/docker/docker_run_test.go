//go:build all || docker
// +build all docker

package docker

import (
	"bufio"
	"context"
	"io"
	"net"
	"strings"
	"testing"
	"time"

	ctypes "github.com/aws/codecatalyst-runner-cli/command-runner/internal/containers/types"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockDockerClient struct {
	client.APIClient
	mock.Mock
}

func (m *mockDockerClient) ContainerExecCreate(ctx context.Context, id string, opts types.ExecConfig) (types.IDResponse, error) {
	args := m.Called(ctx, id, opts)
	return args.Get(0).(types.IDResponse), args.Error(1)
}

func (m *mockDockerClient) ContainerExecAttach(ctx context.Context, id string, opts types.ExecStartCheck) (types.HijackedResponse, error) {
	args := m.Called(ctx, id, opts)
	return args.Get(0).(types.HijackedResponse), args.Error(1)
}

func (m *mockDockerClient) ContainerExecInspect(ctx context.Context, execID string) (types.ContainerExecInspect, error) {
	args := m.Called(ctx, execID)
	return args.Get(0).(types.ContainerExecInspect), args.Error(1)
}

type endlessReader struct {
	io.Reader
}

func (r endlessReader) Read(p []byte) (n int, err error) {
	return 1, nil
}

type mockConn struct {
	net.Conn
	mock.Mock
}

func (m *mockConn) Write(b []byte) (n int, err error) {
	args := m.Called(b)
	return args.Int(0), args.Error(1)
}

func (m *mockConn) Close() (err error) {
	return nil
}

func TestDockerExecAbort(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	conn := &mockConn{}
	conn.On("Write", mock.AnythingOfType("[]uint8")).Return(1, nil)

	client := &mockDockerClient{}
	client.On("ContainerExecCreate", ctx, "123", mock.AnythingOfType("types.ExecConfig")).Return(types.IDResponse{ID: "id"}, nil)
	client.On("ContainerExecAttach", ctx, "id", mock.AnythingOfType("types.ExecStartCheck")).Return(types.HijackedResponse{
		Conn:   conn,
		Reader: bufio.NewReader(endlessReader{}),
	}, nil)

	cr := &dockerContainer{
		id:  "123",
		cli: client,
		input: ctypes.NewContainerInput{
			Image: "image",
		},
	}

	channel := make(chan error)

	go func() {
		channel <- cr.exec([]string{""}, map[string]string{}, "user", "workdir")(ctx)
	}()

	time.Sleep(500 * time.Millisecond)

	cancel()

	err := <-channel
	assert.ErrorIs(t, err, context.Canceled)

	conn.AssertExpectations(t)
	client.AssertExpectations(t)
}

func TestDockerExecFailure(t *testing.T) {
	ctx := context.Background()

	conn := &mockConn{}

	client := &mockDockerClient{}
	client.On("ContainerExecCreate", ctx, "123", mock.AnythingOfType("types.ExecConfig")).Return(types.IDResponse{ID: "id"}, nil)
	client.On("ContainerExecAttach", ctx, "id", mock.AnythingOfType("types.ExecStartCheck")).Return(types.HijackedResponse{
		Conn:   conn,
		Reader: bufio.NewReader(strings.NewReader("output")),
	}, nil)
	client.On("ContainerExecInspect", ctx, "id").Return(types.ContainerExecInspect{
		ExitCode: 1,
	}, nil)

	cr := &dockerContainer{
		id:  "123",
		cli: client,
		input: ctypes.NewContainerInput{
			Image: "image",
		},
	}

	err := cr.exec([]string{""}, map[string]string{}, "user", "workdir")(ctx)
	assert.Error(t, err, "exit with `FAILURE`: 1")

	conn.AssertExpectations(t)
	client.AssertExpectations(t)
}

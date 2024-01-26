package runner

import (
	"bytes"
	"context"
	"fmt"
	"slices"
	"strings"
	"testing"

	cmock "github.com/aws/codecatalyst-runner-cli/command-runner/internal/containers/mock"
	"github.com/aws/codecatalyst-runner-cli/command-runner/internal/containers/types"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type CopyIn struct {
	Source       string
	Target       string
	UseGitIgnore bool
}

func TestExecutorContainer(t *testing.T) {
	type TestParams struct {
		TestCase                 string
		EnvironmentConfiguration EnvironmentConfiguration
		Image                    string
		Entrypoint               Command
		Commands                 []Command
		ExpectedError            string
		ExpectBuild              bool
		ExpectedCopyIn           []CopyIn
	}

	for _, tt := range []*TestParams{
		{
			TestCase: "Basic",
			EnvironmentConfiguration: EnvironmentConfiguration{
				WorkingDir: "testdata/workingdir/basic",
				Env: map[string]string{
					"FOO": "BAR",
				},
			},
			Image:      "docker://alpine:latest",
			Entrypoint: Command{"/bin/cat"},
			Commands: []Command{
				{"/bin/sh", "-c", "echo -n $FOO"},
				{"/bin/sh", "-c", "ls --color=never"},
			},
		},
		{
			TestCase: "Basic dockerfile",
			EnvironmentConfiguration: EnvironmentConfiguration{
				WorkingDir: "testdata/workingdir/basic",
				Env: map[string]string{
					"FOO": "BAR",
				},
			},
			Image:      "../../buildcontext/basic/Dockerfile",
			Entrypoint: Command{"/bin/cat"},
			Commands: []Command{
				{"/bin/sh", "-c", "echo -n $FOO"},
				{"/bin/sh", "-c", "ls --color=never"},
			},
			ExpectBuild: true,
		},
		{
			TestCase: "Basic w/ copy",
			EnvironmentConfiguration: EnvironmentConfiguration{
				WorkingDir: "testdata/workingdir/basic",
				FileMaps: []*FileMap{
					{
						SourcePath: "mock-source",
						TargetPath: "mock-target",
						Type:       FileMapTypeCopyInWithGitignore,
					},
					{
						SourcePath: "mock-source",
						TargetPath: "/abs",
						Type:       FileMapTypeCopyIn,
					},
				},
				Env: map[string]string{
					"FOO": "BAR",
				},
			},
			Image:      "docker://alpine:latest",
			Entrypoint: Command{"/bin/cat"},
			Commands: []Command{
				{"/bin/sh", "-c", "echo -n $FOO"},
			},
			ExpectedCopyIn: []CopyIn{
				{
					Source:       mock.Anything,
					Target:       "/codecatalyst/output/src/mock-target",
					UseGitIgnore: true,
				},
				{
					Source:       mock.Anything,
					Target:       "/abs",
					UseGitIgnore: false,
				},
			},
		},
		{
			TestCase: "Basic w/ bind",
			EnvironmentConfiguration: EnvironmentConfiguration{
				WorkingDir: "testdata/workingdir/basic",
				FileMaps: []*FileMap{
					{
						SourcePath: ".",
						TargetPath: ".",
						Type:       FileMapTypeBind,
					},
				},
				Env: map[string]string{
					"FOO": "BAR",
				},
			},
			Image:      "docker://alpine:latest",
			Entrypoint: Command{"/bin/cat"},
			Commands: []Command{
				{"/bin/sh", "-c", "echo -n $FOO"},
				{"/bin/sh", "-c", "/bin/ls --color=never"},
			},
		},
	} {
		t.Run(tt.TestCase, func(t *testing.T) {
			assert := assert.New(t)
			// setup the code under test
			stdout := new(bytes.Buffer)
			ctx := context.Background()
			mockContainerService := &cmock.MockContainerService{}

			containerWorkingDir := fmt.Sprintf("/codecatalyst/output/src/%s", tt.EnvironmentConfiguration.WorkingDir)
			expectedNewContainerInput := mock.MatchedBy(func(input types.NewContainerInput) bool {
				if input.Stdout != stdout {
					return false
				}
				if input.Name != "codecatalyst-testcontainer" {
					return false
				}
				if !slices.Equal(input.Entrypoint, tt.Entrypoint) {
					return false
				}
				if input.WorkingDir != containerWorkingDir {
					return false
				}
				if !slices.Contains(input.Env, fmt.Sprintf("CATALYST_SOURCE_DIR_WorkflowSource=%s", containerWorkingDir)) {
					return false
				}
				if !slices.Contains(input.Env, fmt.Sprintf("CATALYST_DEFAULT_DIR=%s", containerWorkingDir)) {
					return false
				}
				for k, v := range tt.EnvironmentConfiguration.Env {
					if !slices.Contains(input.Env, fmt.Sprintf("%s=%s", k, v)) {
						return false
					}
				}

				return true
			})

			var emptyMap map[string]string
			var emptyList []string
			scriptMatcher := mock.MatchedBy(func(cmd []string) bool {
				if len(cmd) != 2 {
					return false
				}
				if cmd[0] != "/bin/sh" {
					return false
				}
				if !strings.HasPrefix(cmd[1], "/tmp/mce/tmp/script-") {
					return false
				}
				return true
			})
			mockContainer := &cmock.MockContainer{}
			mockContainer.On("Pull", true).Return(nil)
			mockContainer.On("Exec", []string{"/bin/sh", "/tmp/mce/tmp/envout.sh"}, emptyMap, "", "/").Return(nil)
			mockContainer.On("Remove").Return(nil)
			mockContainer.On("Create", emptyList, emptyList).Return(nil)
			mockContainer.On("Start", false).Return(nil)
			mockContainer.On("Exec", scriptMatcher, emptyMap, "", "").Return(nil)

			mockContainerService.On("NewContainer", expectedNewContainerInput).Return(mockContainer)

			if tt.ExpectBuild {
				mockContainerService.On("ImageExistsLocally", mock.Anything, "codecatalyst-testcontainer:latest", "").Return(true, nil)
				mockContainerService.On("BuildImage", mock.Anything).Return(nil)
			}

			for _, copyIn := range tt.ExpectedCopyIn {
				mockContainer.On("CopyIn", copyIn.Target, copyIn.Source, copyIn.UseGitIgnore).Return(nil)
			}

			envCfg := &tt.EnvironmentConfiguration
			envCfg.Stdout = stdout
			envCfg.Env["CATALYST_SOURCE_DIR_WorkflowSource"] = containerWorkingDir
			executor, err := newContainerCommandExecutor(ctx, &newContainerCommandExecutorParams{
				ID:                       "testcontainer",
				EnvironmentConfiguration: envCfg,
				Image:                    tt.Image,
				Entrypoint:               tt.Entrypoint,
				ContainerService:         mockContainerService,
			})
			if tt.ExpectedError != "" {
				assert.EqualError(err, tt.ExpectedError, "%s - newContainerCommandExecutor()", tt.TestCase)
			} else {
				assert.NoError(err, "%s - newContainerCommandExecutor()", tt.TestCase)
			}

			if executor == nil {
				return
			} else {
				defer executor.Close(false)
			}

			// run the commands
			for _, cmd := range tt.Commands {
				stdout.Reset()

				err = executor.ExecuteCommand(ctx, cmd)
				// assert the results
				assert.NoError(err, "%s - error: %s", tt.TestCase, cmd)
			}
			mockContainerService.AssertExpectations(t)
		})
	}
}

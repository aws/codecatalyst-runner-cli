package runner

import (
	"bytes"
	"context"
	"testing"

	"github.com/kballard/go-shellquote"
	"github.com/stretchr/testify/assert"
)

func TestExecutorShell(t *testing.T) {
	type CommandTest struct {
		Command        string
		ExpectedOutput string
	}

	type TestParams struct {
		TestCase                 string
		EnvironmentConfiguration EnvironmentConfiguration
		Commands                 []CommandTest
		ExpectedError            string
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
			Commands: []CommandTest{
				{
					Command:        "echo -n $FOO",
					ExpectedOutput: "BAR",
				},
				{
					Command:        "ls",
					ExpectedOutput: "main.py\n",
				},
				{
					Command:        "pwd | grep -c testdata/workingdir/basic",
					ExpectedOutput: "1\n",
				},
			},
		},
		{
			TestCase: "Basic w/ clean",
			EnvironmentConfiguration: EnvironmentConfiguration{
				WorkingDir: "testdata/workingdir/basic",
				FileMaps: []*FileMap{
					{
						SourcePath: ".",
						TargetPath: ".",
						Type:       FileMapTypeCopyInWithGitignore,
					},
				},
				Env: map[string]string{
					"FOO": "BAR",
				},
			},
			Commands: []CommandTest{
				{
					Command:        "echo -n $FOO",
					ExpectedOutput: "BAR",
				},
				{
					Command:        "ls",
					ExpectedOutput: "main.py\n",
				},
				{
					Command:        "pwd | grep -v -c testdata/workingdir/basic",
					ExpectedOutput: "1\n",
				},
			},
		},
		{
			TestCase: "Invalid mount",
			EnvironmentConfiguration: EnvironmentConfiguration{
				WorkingDir: "testdata/workingdir/basic",
				FileMaps: []*FileMap{
					{
						SourcePath: "/foo",
						TargetPath: "..",
						Type:       FileMapTypeBind,
					},
				},
			},
			Commands:      []CommandTest{},
			ExpectedError: "unable to use bind mounts with shell executor for non-working directory '/foo'",
		},
	} {
		t.Run(tt.TestCase, func(t *testing.T) {
			assert := assert.New(t)
			// setup the code under test
			stdout := new(bytes.Buffer)
			ctx := context.Background()
			envCfg := &tt.EnvironmentConfiguration
			envCfg.Stdout = stdout
			executor, err := newShellCommandExecutor(ctx, &newShellCommandExecutorParams{
				EnvironmentConfiguration: envCfg,
			})
			if tt.ExpectedError != "" {
				assert.Errorf(err, tt.ExpectedError, "%s - newShellCommandExecutor()", tt.TestCase)
			} else {
				assert.NoError(err, "%s - newShellCommandExecutor()", tt.TestCase)
				defer executor.Close(false)
			}

			// run the commands
			for _, cmd := range tt.Commands {
				stdout.Reset()
				cmdParts, err := shellquote.Split(cmd.Command)
				assert.NoError(err, "%s - shellquote", tt.TestCase)

				err = executor.ExecuteCommand(ctx, cmdParts)
				// assert the results
				assert.NoError(err, "%s - error: %s", tt.TestCase, cmd)
				assert.Equal(cmd.ExpectedOutput, stdout.String(), "%s - output: %s", tt.TestCase, cmd)
			}
		})
	}
}

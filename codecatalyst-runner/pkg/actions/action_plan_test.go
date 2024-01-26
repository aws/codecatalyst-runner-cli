package actions

import (
	"testing"

	"github.com/aws/codecatalyst-runner-cli/command-runner/pkg/runner"

	"github.com/stretchr/testify/assert"
)

func TestNewActionPlan(t *testing.T) {
	type TestParams struct {
		TestCase                         string
		Action                           Action
		ExecutionType                    runner.ExecutionType
		ExpectedEnvironmentConfiguration runner.EnvironmentConfiguration
		ExpectedCommandGroups            []*runner.CommandGroup
		ExpectedErr                      string
	}

	for _, tt := range []*TestParams{
		{
			TestCase: "Empty",
			ExpectedEnvironmentConfiguration: runner.EnvironmentConfiguration{
				Env: map[string]string{
					"CATALYST_WORKFLOW_PROJECT_ID":   "-",
					"CATALYST_WORKFLOW_PROJECT_NAME": "-",
					"CATALYST_WORKFLOW_SPACE_ID":     "-",
					"CATALYST_WORKFLOW_SPACE_NAME":   "-",
					"CI":                             "true",
				},
				FileMaps: []*runner.FileMap{},
			},
			ExpectedCommandGroups: []*runner.CommandGroup{},
		},
		{
			TestCase: "Docker file",
			Action: Action{
				ID:      "test",
				Basedir: "/my/actiondir",
				Runs: Runs{
					Using:         UsingTypeDocker,
					Image:         "Dockerfile",
					PreEntryPoint: "pre-command",
					Entrypoint:    "main-command",
				},
			},
			ExpectedEnvironmentConfiguration: runner.EnvironmentConfiguration{
				Env: map[string]string{
					"CATALYST_WORKFLOW_PROJECT_ID":   "-",
					"CATALYST_WORKFLOW_PROJECT_NAME": "-",
					"CATALYST_WORKFLOW_SPACE_ID":     "-",
					"CATALYST_WORKFLOW_SPACE_NAME":   "-",
					"CI":                             "true",
				},
				FileMaps: []*runner.FileMap{},
			},
			ExpectedCommandGroups: []*runner.CommandGroup{
				{
					Image:      "/my/actiondir/Dockerfile",
					Entrypoint: runner.Command{"/bin/cat"},
					Commands:   []runner.Command{{"pre-command"}},
				},
				{
					Image:      "/my/actiondir/Dockerfile",
					Entrypoint: runner.Command{"/bin/cat"},
					Commands:   []runner.Command{{"main-command"}},
				},
			},
		},
		{
			TestCase: "Docker registry",
			Action: Action{
				ID:      "test",
				Basedir: "/my/actiondir",
				Runs: Runs{
					Using:         UsingTypeDocker,
					Image:         "docker://alpine",
					PreEntryPoint: "pre-command",
					Entrypoint:    "main-command",
				},
			},
			ExpectedEnvironmentConfiguration: runner.EnvironmentConfiguration{
				Env: map[string]string{
					"CATALYST_WORKFLOW_PROJECT_ID":   "-",
					"CATALYST_WORKFLOW_PROJECT_NAME": "-",
					"CATALYST_WORKFLOW_SPACE_ID":     "-",
					"CATALYST_WORKFLOW_SPACE_NAME":   "-",
					"CI":                             "true",
				},
				FileMaps: []*runner.FileMap{},
			},
			ExpectedCommandGroups: []*runner.CommandGroup{
				{
					Image:      "docker://alpine",
					Entrypoint: runner.Command{"/bin/cat"},
					Commands:   []runner.Command{{"pre-command"}},
				},
				{
					Image:      "docker://alpine",
					Entrypoint: runner.Command{"/bin/cat"},
					Commands:   []runner.Command{{"main-command"}},
				},
			},
		},
		{
			TestCase:      "Node12 Shell",
			ExecutionType: runner.ExecutionTypeShell,
			Action: Action{
				ID:      "test",
				Basedir: "/my/actiondir",
				Runs: Runs{
					Using: UsingTypeNode12,
					Pre:   "pre-command",
					Main:  "main-command",
				},
			},
			ExpectedEnvironmentConfiguration: runner.EnvironmentConfiguration{
				Env: map[string]string{
					"CATALYST_WORKFLOW_PROJECT_ID":               "-",
					"CATALYST_WORKFLOW_PROJECT_NAME":             "-",
					"CATALYST_WORKFLOW_SPACE_ID":                 "-",
					"CATALYST_WORKFLOW_SPACE_NAME":               "-",
					"CI":                                         "true",
					"CATALYST_SOURCE_DIR_CawsCustomActionSource": "/my/actiondir",
				},
				FileMaps: []*runner.FileMap{},
			},
			ExpectedCommandGroups: []*runner.CommandGroup{
				{
					Image:      "",
					Entrypoint: runner.Command{},
					Commands: []runner.Command{
						{"node", "/my/actiondir/pre-command"},
					},
				},
				{
					Image:      "",
					Entrypoint: runner.Command{},
					Commands: []runner.Command{
						{"node", "/my/actiondir/main-command"},
					},
				},
			},
		},
		{
			TestCase:      "Node16 Docker",
			ExecutionType: runner.ExecutionTypeDocker,
			Action: Action{
				ID:      "test",
				Basedir: "/my/actiondir",
				Runs: Runs{
					Using: UsingTypeNode16,
					Pre:   "pre-command",
					Main:  "main-command",
				},
			},
			ExpectedEnvironmentConfiguration: runner.EnvironmentConfiguration{
				Env: map[string]string{
					"CATALYST_WORKFLOW_PROJECT_ID":               "-",
					"CATALYST_WORKFLOW_PROJECT_NAME":             "-",
					"CATALYST_WORKFLOW_SPACE_ID":                 "-",
					"CATALYST_WORKFLOW_SPACE_NAME":               "-",
					"CI":                                         "true",
					"CATALYST_SOURCE_DIR_CawsCustomActionSource": "/codecatalyst/output/action",
				},
				FileMaps: []*runner.FileMap{
					{
						SourcePath: "/my/actiondir",
						TargetPath: "/codecatalyst/output/action",
						Type:       runner.FileMapTypeCopyInWithGitignore,
					},
				},
			},
			ExpectedCommandGroups: []*runner.CommandGroup{
				{
					Image:      "docker://public.ecr.aws/codebuild/amazonlinux2-x86_64-standard:5.0",
					Entrypoint: runner.Command{"/bin/cat"},
					Commands: []runner.Command{
						{"node", "/codecatalyst/output/action/test/pre-command"},
					},
				},
				{
					Image:      "docker://public.ecr.aws/codebuild/amazonlinux2-x86_64-standard:5.0",
					Entrypoint: runner.Command{"/bin/cat"},
					Commands: []runner.Command{
						{"node", "/codecatalyst/output/action/test/main-command"},
					},
				},
			},
		},
	} {
		t.Run(tt.TestCase, func(t *testing.T) {
			assert := assert.New(t)
			// setup the code under test
			plan, err := NewActionPlan(&NewActionPlanParams{
				Action:        &tt.Action,
				ExecutionType: tt.ExecutionType,
				WorkingDir:    "/foo",
			})
			if tt.ExpectedErr != "" {
				assert.EqualError(err, tt.ExpectedErr, "%s - Error", tt.TestCase)
			} else {
				assert.NoError(err)
				assert.Equal(tt.Action.ID, plan.ID(), "%s - Plan ID", tt.TestCase)
				assert.Equal(tt.ExpectedEnvironmentConfiguration.Env, plan.EnvironmentConfiguration().Env, "%s - Env", tt.TestCase)
				assert.Equal(tt.ExpectedEnvironmentConfiguration.FileMaps, plan.EnvironmentConfiguration().FileMaps, "%s - FileMaps", tt.TestCase)
				assert.Equal(tt.ExpectedEnvironmentConfiguration.Reuse, plan.EnvironmentConfiguration().Reuse, "%s - Reuse", tt.TestCase)
				assert.Equal("/foo", plan.EnvironmentConfiguration().WorkingDir, "%s - WorkingDir", tt.TestCase)
				assert.Equal(tt.ExpectedCommandGroups, plan.CommandGroups(), "%s - CommandGroups ", tt.TestCase)
			}
		})
	}
}

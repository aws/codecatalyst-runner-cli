package features

import (
	"context"
	"testing"

	"github.com/aws/codecatalyst-runner-cli/command-runner/pkg/runner"

	"github.com/stretchr/testify/assert"
)

func TestWorkingDirImporter(t *testing.T) {
	cacheDir := "/tmp/cache"

	type TestParams struct {
		TestCase          string
		WorkingDir        string
		BindWorkingDir    bool
		SharedCompute     bool
		IsWorkingDirSetup bool
		ExpectedFileMap   []*runner.FileMap
	}

	for _, tt := range []*TestParams{
		{
			TestCase:       "Copy WorkingDir",
			WorkingDir:     "/foo/bar",
			BindWorkingDir: false,
			SharedCompute:  false,
			ExpectedFileMap: []*runner.FileMap{
				{
					SourcePath: "/foo/bar/.",
					TargetPath: "git/v1/bar",
					Type:       runner.FileMapTypeCopyInWithGitignore,
				},
			},
		},
		{
			TestCase:       "First Copy WorkingDir w/Shared Compute",
			WorkingDir:     "/foo/bar",
			BindWorkingDir: false,
			SharedCompute:  true,
			ExpectedFileMap: []*runner.FileMap{
				{
					SourcePath: "/foo/bar/.",
					TargetPath: "git/v1/bar",
					Type:       runner.FileMapTypeCopyInWithGitignore,
				},
				{
					SourcePath: "git/v1/bar/.",
					TargetPath: "/tmp/cache/sources/WorkflowSource",
					Type:       runner.FileMapTypeCopyOut,
				},
			},
		},
		{
			TestCase:          "Subsequent Copy WorkingDir w/Shared Compute",
			WorkingDir:        "/foo/bar",
			BindWorkingDir:    false,
			SharedCompute:     true,
			IsWorkingDirSetup: true,
			ExpectedFileMap: []*runner.FileMap{
				{
					SourcePath: "/tmp/cache/sources/WorkflowSource/.",
					TargetPath: "git/v1/bar",
					Type:       runner.FileMapTypeCopyInWithGitignore,
				},
				{
					SourcePath: "git/v1/bar/.",
					TargetPath: "/tmp/cache/sources/WorkflowSource",
					Type:       runner.FileMapTypeCopyOut,
				},
			},
		},
		{
			TestCase:       "Bind WorkingDir",
			WorkingDir:     "/foo/bar",
			BindWorkingDir: true,
			SharedCompute:  false,
			ExpectedFileMap: []*runner.FileMap{
				{
					SourcePath: "/foo/bar",
					TargetPath: "git/v1/bar",
					Type:       runner.FileMapTypeBind,
				},
			},
		},
	} {
		t.Run(tt.TestCase, func(t *testing.T) {
			assert := assert.New(t)
			// setup the code under test
			ctx := context.Background()
			feature := WorkingDirImporter(tt.WorkingDir, cacheDir, tt.BindWorkingDir, tt.SharedCompute, &tt.IsWorkingDirSetup)

			// setup the mock
			plan := new(runner.MockPlan)
			m := new(runner.MockPlanExecutor).WithPlan(plan)
			m.OnExecute(ctx).Return(nil)

			// run the feature
			err := m.Execute(ctx, feature)

			// assert the results
			assert.NoError(err)
			m.AssertExpectations(t)
			assert.Equal(tt.ExpectedFileMap, plan.EnvironmentConfiguration().FileMaps, "%s - FileMap ", tt.TestCase)
		})
	}
}

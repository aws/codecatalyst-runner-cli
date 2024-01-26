package workflows

import (
	"context"
	"fmt"
	"testing"

	"github.com/aws/codecatalyst-runner-cli/command-runner/pkg/runner"

	"github.com/stretchr/testify/assert"
)

func TestOutputArtifacts(t *testing.T) {
	type TestParams struct {
		TestCase              string
		PlanID                string
		OutputArtifacts       []*OutputArtifact
		ExpectedArtifactPlans map[string]string
		ExpectedFileMaps      []*runner.FileMap
	}

	mockCacheDir := "/cachedir"

	for _, tt := range []*TestParams{
		{
			TestCase: "basic-single-file",
			PlanID:   "SomeAction",
			OutputArtifacts: []*OutputArtifact{
				{
					Name:  "ARTIFACT1",
					Files: "out/file.txt",
				},
			},
			ExpectedArtifactPlans: map[string]string{
				"ARTIFACT1": "SomeAction",
			},
			ExpectedFileMaps: []*runner.FileMap{
				{
					SourcePath: "git/v1/myapp/out/file.txt",
					TargetPath: fmt.Sprintf("%s/artifacts/ARTIFACT1/out/file.txt", mockCacheDir),
					Type:       runner.FileMapTypeCopyOut,
				},
			},
		},
		{
			TestCase: "basic-single-directory",
			PlanID:   "SomeAction",
			OutputArtifacts: []*OutputArtifact{
				{
					Name:  "ARTIFACT1",
					Files: "out/*",
				},
			},
			ExpectedArtifactPlans: map[string]string{
				"ARTIFACT1": "SomeAction",
			},
			ExpectedFileMaps: []*runner.FileMap{
				{
					SourcePath: "git/v1/myapp/out/*",
					TargetPath: fmt.Sprintf("%s/artifacts/ARTIFACT1/out/", mockCacheDir),
					Type:       runner.FileMapTypeCopyOut,
				},
			},
		},
		{
			TestCase: "glob-all",
			PlanID:   "SomeAction",
			OutputArtifacts: []*OutputArtifact{
				{
					Name:  "ARTIFACT1",
					Files: "**/*",
				},
			},
			ExpectedArtifactPlans: map[string]string{
				"ARTIFACT1": "SomeAction",
			},
			ExpectedFileMaps: []*runner.FileMap{
				{
					SourcePath: "git/v1/myapp/**/*",
					TargetPath: fmt.Sprintf("%s/artifacts/ARTIFACT1/", mockCacheDir),
					Type:       runner.FileMapTypeCopyOut,
				},
			},
		},
		{
			TestCase: "glob-subdir",
			PlanID:   "SomeAction",
			OutputArtifacts: []*OutputArtifact{
				{
					Name:  "ARTIFACT1",
					Files: "out/**/*",
				},
			},
			ExpectedArtifactPlans: map[string]string{
				"ARTIFACT1": "SomeAction",
			},
			ExpectedFileMaps: []*runner.FileMap{
				{
					SourcePath: "git/v1/myapp/out/**/*",
					TargetPath: fmt.Sprintf("%s/artifacts/ARTIFACT1/out/", mockCacheDir),
					Type:       runner.FileMapTypeCopyOut,
				},
			},
		},
		{
			TestCase: "basic-multiple-file",
			PlanID:   "SomeAction",
			OutputArtifacts: []*OutputArtifact{
				{
					Name: "ARTIFACT1",
					Files: []string{
						"out/file.txt",
						"out/subdir",
					},
				},
			},
			ExpectedArtifactPlans: map[string]string{
				"ARTIFACT1": "SomeAction",
			},
			ExpectedFileMaps: []*runner.FileMap{
				{
					SourcePath: "git/v1/myapp/out/file.txt",
					TargetPath: fmt.Sprintf("%s/artifacts/ARTIFACT1/out/file.txt", mockCacheDir),
					Type:       runner.FileMapTypeCopyOut,
				},
				{
					SourcePath: "git/v1/myapp/out/subdir",
					TargetPath: fmt.Sprintf("%s/artifacts/ARTIFACT1/out/subdir", mockCacheDir),
					Type:       runner.FileMapTypeCopyOut,
				},
			},
		},
	} {
		t.Run(tt.TestCase, func(t *testing.T) {
			assert := assert.New(t)
			// setup the code under test
			ctx := context.Background()
			artifactPlans := make(map[string]string)
			feature := OutputArtifacts(tt.PlanID, tt.OutputArtifacts, artifactPlans, mockCacheDir)

			// setup the mock
			plan := new(runner.MockPlan)
			plan.WithID(tt.PlanID)
			plan.EnvironmentConfiguration().WorkingDir = "/home/user/myapp"
			m := new(runner.MockPlanExecutor).WithPlan(plan)
			m.OnExecute(ctx).Return(nil)

			// run the feature
			err := m.Execute(ctx, feature)

			assert.NoError(err)
			assert.Equal(tt.ExpectedFileMaps, plan.EnvironmentConfiguration().FileMaps)
			assert.Equal(tt.ExpectedArtifactPlans, artifactPlans)

			// assert the results
			m.AssertExpectations(t)
		})
	}
}

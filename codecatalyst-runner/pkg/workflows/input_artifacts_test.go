package workflows

import (
	"context"
	"fmt"
	"testing"

	"github.com/aws/codecatalyst-runner-cli/command-runner/pkg/runner"

	"github.com/stretchr/testify/assert"
)

func TestInputArtifacts(t *testing.T) {
	type TestParams struct {
		TestCase          string
		DependsOn         []string
		InputArtifacts    []string
		ArtifactPlans     map[string]string
		ExpectedEnv       map[string]string
		ExpectedFileMaps  []*runner.FileMap
		ExpectedDependsOn []string
		ExpectedError     string
	}

	mockCacheDir := "/cachedir"

	for _, tt := range []*TestParams{
		{
			TestCase:       "basic",
			InputArtifacts: []string{"Artifact1"},
			ArtifactPlans: map[string]string{
				"Artifact1": "SomeAction",
			},
			DependsOn:         []string{"SomeAction"},
			ExpectedDependsOn: []string{"SomeAction"},
			ExpectedEnv: map[string]string{
				"CATALYST_SOURCE_DIR_Artifact1": "s3/01",
			},
			ExpectedFileMaps: []*runner.FileMap{
				{
					SourcePath: fmt.Sprintf("%s/artifacts/Artifact1/.", mockCacheDir),
					TargetPath: "s3/01",
					Type:       runner.FileMapTypeCopyIn,
				},
			},
		},
		{
			TestCase:       "basic-without-depends-on",
			InputArtifacts: []string{"Artifact1"},
			ArtifactPlans: map[string]string{
				"Artifact1": "SomeAction",
			},
			ExpectedDependsOn: []string{"SomeAction"},
			ExpectedError:     "deferred",
		},
		{
			TestCase:       "basic-without-plan",
			InputArtifacts: []string{"Artifact1"},
			ExpectedError:  "deferred",
		},
	} {
		t.Run(tt.TestCase, func(t *testing.T) {
			assert := assert.New(t)
			// setup the code under test
			ctx := context.Background()
			feature := InputArtifacts(tt.InputArtifacts, tt.ArtifactPlans, mockCacheDir)

			// setup the mock
			plan := new(runner.MockPlan)
			plan.AddDependsOn(tt.DependsOn...)
			m := new(runner.MockPlanExecutor).WithPlan(plan)
			if tt.ExpectedError == "" {
				m.OnExecute(ctx).Return(nil)
			}

			// run the feature
			err := m.Execute(ctx, feature)

			if tt.ExpectedError != "" {
				assert.Error(err)
				assert.EqualError(err, tt.ExpectedError)
			} else {
				assert.NoError(err)
				assert.Equal(tt.ExpectedEnv, plan.EnvironmentConfiguration().Env)
				assert.Equal(tt.ExpectedFileMaps, plan.EnvironmentConfiguration().FileMaps)
				assert.Equal(tt.ExpectedDependsOn, plan.DependsOn())
			}

			// assert the results
			m.AssertExpectations(t)
		})
	}
}

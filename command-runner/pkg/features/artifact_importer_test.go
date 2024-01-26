package features

import (
	"context"
	"testing"

	"github.com/aws/codecatalyst-runner-cli/command-runner/pkg/runner"

	"github.com/stretchr/testify/assert"
)

func TestArtifactImporter(t *testing.T) {
	assert := assert.New(t)

	type TestParams struct {
		TestCase         string
		ArtifactDir      string
		Artifacts        []string
		ExpectedFileMaps []*runner.FileMap
	}

	for _, tt := range []*TestParams{
		{
			TestCase:    "Single Artifact",
			ArtifactDir: "/foo/bar",
			Artifacts:   []string{"baz"},
			ExpectedFileMaps: []*runner.FileMap{
				{
					SourcePath: "/foo/bar/baz",
					TargetPath: "./",
					Type:       runner.FileMapTypeCopyIn,
				},
			},
		},
	} {
		// setup the code under test
		ctx := context.Background()
		feature := ArtifactImporter(tt.ArtifactDir, tt.Artifacts...)

		// setup the mock
		plan := new(runner.MockPlan)
		m := new(runner.MockPlanExecutor).WithPlan(plan)
		m.OnExecute(ctx).Return(nil)

		// run the feature
		err := m.Execute(ctx, feature)

		// assert the results
		assert.NoError(err)
		m.AssertExpectations(t)
		assert.Len(plan.EnvironmentConfiguration().FileMaps, len(tt.ExpectedFileMaps), "%s - FileMap len", tt.TestCase)
		assert.Equal(tt.ExpectedFileMaps, plan.EnvironmentConfiguration().FileMaps, "%s - FileMaps", tt.TestCase)
	}
}

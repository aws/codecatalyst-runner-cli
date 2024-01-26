package features

import (
	"context"
	"testing"

	"github.com/aws/codecatalyst-runner-cli/command-runner/pkg/runner"

	"github.com/stretchr/testify/assert"
)

func TestSBOMFeature(t *testing.T) {
	assert := assert.New(t)

	type TestParams struct {
		TestCase     string
		ReportDir    string
		ExpectedType SBOMType
	}

	for _, tt := range []*TestParams{
		{
			TestCase:     "Empty report",
			ReportDir:    "testdata/reports/empty",
			ExpectedType: "",
		},
		{
			TestCase:     "XML report",
			ReportDir:    "testdata/reports/non-sarif",
			ExpectedType: "",
		},
		{
			TestCase:     "Simple SBOM",
			ReportDir:    "testdata/reports/spdx",
			ExpectedType: SBOMTypeSPDX,
		},
	} {
		// setup the code under test
		ctx := context.Background()
		sbom := new(SBOM)
		feature := SBOMDetector(tt.ReportDir, sbom)

		// setup the mock
		m := new(runner.MockPlanExecutor)
		m.OnExecute(ctx).Return(nil)

		// run the feature
		err := m.Execute(ctx, feature)

		// assert the results
		assert.NoError(err)
		m.AssertExpectations(t)
		assert.NoError(err)
		assert.Equal(tt.ExpectedType, sbom.Type, "%s - Type", tt.TestCase)
	}
}

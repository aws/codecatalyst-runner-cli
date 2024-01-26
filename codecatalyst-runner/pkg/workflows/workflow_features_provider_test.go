package workflows

import (
	"testing"

	"github.com/aws/codecatalyst-runner-cli/command-runner/pkg/runner"

	"github.com/stretchr/testify/assert"
)

func TestNewActionFeaturesProvider(t *testing.T) {
	assert := assert.New(t)

	// setup
	mockWorkflow := new(Workflow)
	featuresProvider, err := NewWorkflowFeaturesProvider(&NewWorkflowFeaturesProviderParams{
		EnvironmentConfiguration: runner.EnvironmentConfiguration{},
		Workflow:                 mockWorkflow,
	})

	if err != nil {
		assert.NoError(err)
	} else {
		mp := new(runner.MockPlan).WithID("test1")
		features, err := featuresProvider.Features(mp)
		assert.NoError(err)
		assert.Len(features, 8)
	}
}

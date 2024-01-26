package features

import (
	"context"
	"testing"

	"github.com/aws/codecatalyst-runner-cli/command-runner/pkg/runner"

	"github.com/stretchr/testify/assert"
)

func TestLogDurationFeature(t *testing.T) {
	assert := assert.New(t)
	ctx := context.Background()

	// setup the code under test
	feature := TracePlan()

	// setup the mock
	mp := new(runner.MockPlan)
	m := new(runner.MockPlanExecutor).WithPlan(mp)
	m.OnExecute(ctx).Return(nil)

	// run the feature
	err := m.Execute(ctx, feature)
	assert.NoError(err)

	m.AssertExpectations(t)
}

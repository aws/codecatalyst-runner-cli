package features

import (
	"context"
	"testing"

	"github.com/aws/codecatalyst-runner-cli/command-runner/pkg/runner"

	"github.com/stretchr/testify/assert"
)

func TestReuseDisabled(t *testing.T) {
	assert := assert.New(t)

	// setup the code under test
	ctx := context.Background()
	feature := Reuse(false)

	// setup the mock
	mp := new(runner.MockPlan)
	m := new(runner.MockPlanExecutor).WithPlan(mp)
	m.OnExecute(ctx).Return(nil)

	// run the feature
	err := m.Execute(ctx, feature)

	// assert the results
	assert.NoError(err)
	m.AssertExpectations(t)
	assert.False(mp.EnvironmentConfiguration().Reuse)
}

func TeturestReuseEnabled(t *testing.T) {
	assert := assert.New(t)

	// setup the code under test
	ctx := context.Background()
	feature := Reuse(true)

	// setup the mock
	mp := new(runner.MockPlan)
	m := new(runner.MockPlanExecutor).WithPlan(mp)
	m.OnExecute(ctx).Return(nil)

	// run the feature
	err := m.Execute(ctx, feature)

	// assert the results
	assert.NoError(err)
	m.AssertExpectations(t)
	assert.True(mp.EnvironmentConfiguration().Reuse)
}

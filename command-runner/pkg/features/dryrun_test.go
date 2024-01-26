package features

import (
	"context"
	"fmt"
	"testing"

	"github.com/aws/codecatalyst-runner-cli/command-runner/pkg/common"
	"github.com/aws/codecatalyst-runner-cli/command-runner/pkg/runner"

	"github.com/stretchr/testify/assert"
)

func TestDryRunDisabled(t *testing.T) {
	assert := assert.New(t)

	// setup the code under test
	ctx := context.Background()
	feature := Dryrun(false)

	// setup the mock
	m := new(runner.MockPlanExecutor)
	m.OnExecute(ctx).Return(nil)

	// run the feature
	err := m.Execute(ctx, feature)

	// assert the results
	assert.NoError(err, "Dryrun should not return an error")
	m.AssertExpectations(t)
}

func TestDryRunDisabledWithError(t *testing.T) {
	assert := assert.New(t)

	// setup the code under test
	ctx := context.Background()
	feature := Dryrun(false)

	// setup the mock
	m := new(runner.MockPlanExecutor)
	m.OnExecute(ctx).Return(fmt.Errorf("mock error"))

	// run the feature
	err := m.Execute(ctx, feature)

	// assert the results
	assert.EqualError(err, "mock error")
	m.AssertExpectations(t)
}

func TestDryRunEnabled(t *testing.T) {
	assert := assert.New(t)

	// setup the code under test
	ctx := context.Background()
	feature := Dryrun(true)

	// setup the mock
	m := new(runner.MockPlanExecutor)

	// run the feature
	err := m.Execute(ctx, feature)

	// assert the results
	assert.NoError(err, "Dryrun should not return an error")
	m.AssertExpectations(t)
	m.AssertNotCalled(t, "Execute")
}

func TestDryRunEnabledInContext(t *testing.T) {
	assert := assert.New(t)

	// setup the code under test
	ctx := common.WithDryrun(context.Background(), true)
	feature := Dryrun(true)

	// setup the mock
	m := new(runner.MockPlanExecutor)

	// run the feature
	err := m.Execute(ctx, feature)

	// assert the results
	assert.NoError(err, "Dryrun should not return an error")
	m.AssertExpectations(t)
	m.AssertNotCalled(t, "Execute")
}

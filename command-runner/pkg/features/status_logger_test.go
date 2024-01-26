package features

import (
	"context"
	"fmt"
	"testing"

	"github.com/aws/codecatalyst-runner-cli/command-runner/pkg/common"
	"github.com/aws/codecatalyst-runner-cli/command-runner/pkg/runner"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestStatusLoggerSuccess(t *testing.T) {
	assert := assert.New(t)

	// setup the code under test
	ctx := context.Background()
	feature := StatusLogger("mock-context")

	// setup the mock
	m := new(runner.MockPlanExecutor)
	m.OnExecute(mock.MatchedBy(func(ctx context.Context) bool {
		return true
	})).Return(nil)

	// run the feature
	err := m.Execute(ctx, feature)

	// assert the results
	assert.NoError(err)
	m.AssertExpectations(t)
}

func TestStatusLoggerFail(t *testing.T) {
	assert := assert.New(t)

	// setup the code under test
	ctx := context.Background()
	feature := StatusLogger("mock-context")

	// setup the mock
	m := new(runner.MockPlanExecutor)
	m.OnExecute(mock.MatchedBy(func(ctx context.Context) bool {
		return true
	})).Return(fmt.Errorf("mock-error"))

	// run the feature
	err := m.Execute(ctx, feature)

	// assert the results
	assert.EqualError(err, "mock-error")
	m.AssertExpectations(t)
}

func TestStatusLoggerWarn(t *testing.T) {
	assert := assert.New(t)

	// setup the code under test
	ctx := context.Background()
	feature := StatusLogger("mock-context")

	// setup the mock
	m := new(runner.MockPlanExecutor)
	m.OnExecute(mock.MatchedBy(func(ctx context.Context) bool {
		return true
	})).Return(common.NewWarning("mock-warning"))

	// run the feature
	err := m.Execute(ctx, feature)

	// assert the results
	assert.NoError(err)
	m.AssertExpectations(t)
}

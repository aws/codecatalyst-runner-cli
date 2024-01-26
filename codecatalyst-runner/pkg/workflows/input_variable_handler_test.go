package workflows

import (
	"context"
	"testing"

	"github.com/aws/codecatalyst-runner-cli/command-runner/pkg/runner"

	"github.com/stretchr/testify/assert"
)

func TestInputVariableHandler(t *testing.T) {
	type TestParams struct {
		TestCase    string
		Inputs      map[string]string
		ExpectedEnv map[string]string
	}

	for _, tt := range []*TestParams{
		{
			TestCase: "basic",
			Inputs: map[string]string{
				"Hello": "Alice",
			},
			ExpectedEnv: map[string]string{
				"Hello": "Alice",
			},
		},
	} {
		t.Run(tt.TestCase, func(t *testing.T) {
			assert := assert.New(t)
			// setup the code under test
			ctx := context.Background()
			feature := InputVariableHandler(tt.Inputs)

			// setup the mock
			plan := new(runner.MockPlan)
			m := new(runner.MockPlanExecutor).WithPlan(plan)
			m.OnExecute(ctx).Return(nil)

			// run the feature
			err := m.Execute(ctx, feature)

			assert.NoError(err)
			assert.Equal(tt.ExpectedEnv, plan.EnvironmentConfiguration().Env)

			// assert the results
			m.AssertExpectations(t)
		})
	}
}

package actions

import (
	"context"
	"testing"

	"github.com/aws/codecatalyst-runner-cli/command-runner/pkg/runner"

	"github.com/stretchr/testify/assert"
)

func TestActionInputHandler(t *testing.T) {
	assert := assert.New(t)

	type TestParams struct {
		TestCase            string
		Inputs              map[string]string
		ActionConfiguration map[string]Parameter
		ExpectedEnv         map[string]string
		ExpectedError       string
	}

	for _, tt := range []*TestParams{
		{
			TestCase:    "Nil inputs",
			Inputs:      nil,
			ExpectedEnv: nil,
		},
		{
			TestCase:    "Empty inputs",
			Inputs:      map[string]string{},
			ExpectedEnv: map[string]string{},
		},
		{
			TestCase: "Extra inputs",
			Inputs:   map[string]string{"Foo": "Bar", "Baz": "Qux"},
			ActionConfiguration: map[string]Parameter{
				"Foo": {},
			},
			ExpectedEnv: map[string]string{"INPUT_FOO": "Bar"},
		},
		{
			TestCase: "Default inputs",
			Inputs:   map[string]string{},
			ActionConfiguration: map[string]Parameter{
				"Foo": {
					Default: "Bar",
				},
			},
			ExpectedEnv: map[string]string{"INPUT_FOO": "Bar"},
		},
		{
			TestCase: "Missing inputs",
			Inputs:   map[string]string{},
			ActionConfiguration: map[string]Parameter{
				"Foo": {
					Required: true,
				},
			},
			ExpectedError: "input parameter 'Foo' is required for action 'Missing inputs'",
		},
	} {
		// setup the code under test
		ctx := context.Background()
		feature := ActionInputHandler(tt.Inputs)

		// setup the mock
		plan := new(actionPlan)
		plan.environmentConfiguration = &runner.EnvironmentConfiguration{}
		plan.action = &Action{
			ID:            tt.TestCase,
			Configuration: tt.ActionConfiguration,
		}
		m := new(runner.MockPlanExecutor).WithPlan(plan)
		m.OnExecute(ctx).Return(nil)

		// run the feature
		err := m.Execute(ctx, feature)

		// assert the results
		if tt.ExpectedError != "" {
			assert.EqualError(err, tt.ExpectedError)
		} else {
			assert.NoError(err)
			m.AssertExpectations(t)
			assert.Equal(tt.ExpectedEnv, plan.EnvironmentConfiguration().Env, "%s - Env", tt.TestCase)
		}
	}
}

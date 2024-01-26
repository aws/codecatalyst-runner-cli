package actions

import (
	"context"
	"fmt"
	"testing"

	"github.com/aws/codecatalyst-runner-cli/command-runner/pkg/runner"

	"github.com/stretchr/testify/assert"
)

func TestActionOutputHandler(t *testing.T) {
	type TestParams struct {
		TestCase              string
		SuppressOutput        bool
		Stdouts               []string
		ActionOutputVariables map[string]Output
		ExpectedOutputs       map[string]string
		ExpectedError         string
	}

	for _, tt := range []*TestParams{
		{
			TestCase:              "No outputs",
			ActionOutputVariables: map[string]Output{},
			ExpectedOutputs:       map[string]string{},
		},
		{
			TestCase: "Missing outputs",
			ActionOutputVariables: map[string]Output{
				"Foo": {},
			},
			ExpectedOutputs: map[string]string{},
		},
		{
			TestCase: "Valid outputs",
			ActionOutputVariables: map[string]Output{
				"Foo": {},
			},
			Stdouts: []string{
				"::set-output name=Foo::Bar",
			},
			ExpectedOutputs: map[string]string{
				"Foo": "Bar",
			},
		},
		{
			TestCase: "Valid outputs with suppress",
			ActionOutputVariables: map[string]Output{
				"Foo": {},
			},
			SuppressOutput: true,
			Stdouts: []string{
				"::set-output name=Foo::Bar",
			},
			ExpectedOutputs: map[string]string{
				"Foo": "Bar",
			},
		},
		{
			TestCase:              "Invalid outputs",
			ActionOutputVariables: map[string]Output{},
			Stdouts: []string{
				"::set-output name=Foo::Bar",
			},
			ExpectedOutputs: map[string]string{},
		},
		{
			TestCase:              "ActionRunSummarySuccess",
			ActionOutputVariables: map[string]Output{},
			Stdouts: []string{
				"::set-output name=ACTION_RUN_SUMMARY::[{}]",
			},
			ExpectedOutputs: map[string]string{},
		},
		{
			TestCase:              "ActionRunSummaryError",
			ActionOutputVariables: map[string]Output{},
			Stdouts: []string{
				"::set-output name=ACTION_RUN_SUMMARY::[{\"Level\":\"Error\",\"Text\":\"MyError\",\"Message\":\"test error\"}]",
			},
			ExpectedError: "[MyError] test error",
		},
	} {
		t.Run(tt.TestCase, func(t *testing.T) {
			assert := assert.New(t)
			// setup the code under test
			ctx := context.Background()
			outputs := make(map[string]string)
			feature := ActionOutputHandler(outputs, tt.SuppressOutput)

			// setup the mock
			plan := new(actionPlan)
			plan.environmentConfiguration = &runner.EnvironmentConfiguration{}
			plan.action = &Action{
				ID: tt.TestCase,
				Outputs: Outputs{
					Variables: tt.ActionOutputVariables,
				},
			}
			m := new(runner.MockPlanExecutor).WithPlan(plan).WithExecutor(func(ctx context.Context) error {
				for _, msg := range tt.Stdouts {
					fmt.Fprintln(plan.environmentConfiguration.Stdout, msg)
				}
				return nil
			})
			m.OnExecute(ctx).Return(nil)

			// run the feature
			err := m.Execute(ctx, feature)

			// assert the results
			if tt.ExpectedError != "" {
				assert.EqualError(err, tt.ExpectedError)
			} else {
				assert.NoError(err)
				m.AssertExpectations(t)
				assert.Equal(tt.ExpectedOutputs, outputs, "%s - Outputs", tt.TestCase)
			}
		})
	}
}

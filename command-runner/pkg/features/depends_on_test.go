package features

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/aws/codecatalyst-runner-cli/command-runner/pkg/runner"

	"github.com/stretchr/testify/assert"
)

func TestDependsOn(t *testing.T) {
	for _, tt := range []struct {
		TestCase         string
		SuccessDependsOn []string
		FailureDependsOn []string
		PendingDependsOn []string
		Error            string
		ExpectedError    string
	}{
		{
			TestCase: "no-dependencies",
		},
		{
			TestCase:         "succeess-dependencies",
			SuccessDependsOn: []string{"success-id"},
		},
		{
			TestCase:         "failed-dependencies",
			FailureDependsOn: []string{"failed-id"},
			ExpectedError:    "cancelled failed-dependencies: dependency failed-id failed",
		},
		{
			TestCase:         "pending-dependencies",
			PendingDependsOn: []string{"pending-id"},
			ExpectedError:    "deferred",
		},
		{
			TestCase:      "handle-error",
			Error:         "mock error",
			ExpectedError: "mock error",
		},
	} {
		t.Run(tt.TestCase, func(t *testing.T) {
			assert := assert.New(t)
			planTracker := new(PlanTracker)

			// setup the code under test
			ctx := context.Background()
			mockPlan := &runner.MockPlan{}
			mockPlan = mockPlan.WithID(tt.TestCase)
			feature := DependsOn(planTracker.ProgressHandle(mockPlan.ID()))

			for _, id := range tt.PendingDependsOn {
				mockPlan.AddDependsOn(id)
				planTracker.ProgressHandle(id)
			}
			for _, id := range tt.SuccessDependsOn {
				mockPlan.AddDependsOn(id)
				planTracker.ProgressHandle(id).Success()
			}
			for _, id := range tt.FailureDependsOn {
				mockPlan.AddDependsOn(id)
				planTracker.ProgressHandle(id).Failure(fmt.Errorf("failed"))
			}

			// setup the mock
			m := new(runner.MockPlanExecutor)
			m.WithPlan(mockPlan)

			// only mock 'Execute()' if all dependencies are met
			if len(tt.FailureDependsOn)+len(tt.PendingDependsOn) == 0 {
				var returnErr error
				if tt.Error != "" {
					returnErr = errors.New(tt.Error)
				}
				m.OnExecute(ctx).Return(returnErr)
			}

			// run the feature
			err := m.Execute(ctx, feature)

			// assert the results
			if tt.ExpectedError != "" {
				assert.EqualError(err, tt.ExpectedError, "DependsOn should return the error")
			} else {
				assert.NoError(err, "DependsOn should not return an error")
			}
			m.AssertExpectations(t)
		})
	}
}

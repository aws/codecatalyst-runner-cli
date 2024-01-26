package features

import (
	"context"
	"fmt"
	"runtime/trace"

	"github.com/aws/codecatalyst-runner-cli/command-runner/pkg/runner"
)

// TracePlan is a feature to trace the runtime of an action
func TracePlan() runner.Feature {
	return func(ctx context.Context, plan runner.Plan, e runner.PlanExecutor) error {
		region := trace.StartRegion(ctx, fmt.Sprintf("plan-%s", plan.ID()))
		defer region.End()
		return e(ctx)
	}
}

package features

import (
	"context"

	"github.com/aws/codecatalyst-runner-cli/command-runner/pkg/runner"

	"github.com/rs/zerolog/log"
)

// Reuse provides a Feature to configure whether a plan is executed with reused containers
func Reuse(reuse bool) runner.Feature {
	return func(ctx context.Context, plan runner.Plan, e runner.PlanExecutor) error {
		log.Ctx(ctx).Debug().Msgf("ENTER Reuse %v", reuse)
		plan.EnvironmentConfiguration().Reuse = reuse
		err := e(ctx)
		log.Ctx(ctx).Debug().Msg("EXIT Reuse")
		return err
	}
}

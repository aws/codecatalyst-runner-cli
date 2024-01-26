package features

import (
	"context"

	"github.com/aws/codecatalyst-runner-cli/command-runner/pkg/common"
	"github.com/aws/codecatalyst-runner-cli/command-runner/pkg/runner"

	"github.com/rs/zerolog/log"
)

// StatusLogger is a Feature to configure logging of the result of a plan execution
func StatusLogger(contextName string) runner.Feature {
	return func(ctx context.Context, plan runner.Plan, e runner.PlanExecutor) error {
		log.Ctx(ctx).Debug().Msg("ENTER StatusLogger")
		log.Ctx(ctx).Info().Msg("✨ STARTING")
		err := e(ctx)
		if err == nil {
			log.Ctx(ctx).Info().Msg("✅ SUCCESS")
		} else {
			switch err.(type) {
			case common.Warning:
				log.Ctx(ctx).Warn().Msgf("   %s", err.Error())
				err = nil
			default:
				log.Ctx(ctx).Error().Err(err).Msg("❌ FAILED")
			}
		}
		log.Ctx(ctx).Debug().Msg("EXIT StatusLogger")
		return err
	}
}

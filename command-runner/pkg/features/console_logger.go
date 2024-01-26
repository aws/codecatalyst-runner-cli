package features

import (
	"context"
	"os"

	"github.com/aws/codecatalyst-runner-cli/command-runner/pkg/runner"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// ConsoleLogger is a Feature to capture simple logs
func ConsoleLogger() runner.Feature {
	return func(ctx context.Context, plan runner.Plan, e runner.PlanExecutor) error {
		log.Ctx(ctx).Debug().Msg("ENTER ConsoleLogger")
		ctx = log.Logger.
			Output(zerolog.ConsoleWriter{Out: os.Stdout}).
			With().
			Str("id", plan.ID()).
			Logger().
			WithContext(ctx)
		err := e(ctx)
		log.Ctx(ctx).Debug().Msg("EXIT ConsoleLogger")
		return err
	}
}

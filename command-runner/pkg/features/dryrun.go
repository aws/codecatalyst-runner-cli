package features

import (
	"context"

	"github.com/aws/codecatalyst-runner-cli/command-runner/pkg/common"
	"github.com/aws/codecatalyst-runner-cli/command-runner/pkg/runner"

	"github.com/rs/zerolog/log"
)

// Dryrun is a Feature to skip execution when dryrun is true
func Dryrun(dryrun bool) runner.Feature {
	return func(ctx context.Context, plan runner.Plan, e runner.PlanExecutor) error {
		log.Ctx(ctx).Debug().Msg("ENTER Dryrun")
		if dryrun || common.Dryrun(ctx) {
			log.Ctx(ctx).Debug().Msg("EXIT Dryrun early")
			return nil
		}
		err := e(ctx)
		log.Ctx(ctx).Debug().Msg("EXIT Dryrun")
		return err
	}
}

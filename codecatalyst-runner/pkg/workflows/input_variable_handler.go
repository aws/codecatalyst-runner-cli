package workflows

import (
	"context"

	"github.com/aws/codecatalyst-runner-cli/command-runner/pkg/runner"

	"github.com/rs/zerolog/log"
)

// InputVariableHandler converts the provided inputs into environment variables
func InputVariableHandler(inputs map[string]string) runner.Feature {
	return func(ctx context.Context, plan runner.Plan, e runner.PlanExecutor) error {
		log.Ctx(ctx).Debug().Msg("ENTER VariableHandler")
		if inputs == nil {
			log.Ctx(ctx).Debug().Msg("SKIP VariableHandler - inputs == nil")
		} else {
			envCfg := plan.EnvironmentConfiguration()
			if envCfg.Env == nil {
				envCfg.Env = make(map[string]string)
			}
			for name, value := range inputs {
				envCfg.Env[name] = value
			}
		}

		err := e(ctx)
		log.Ctx(ctx).Debug().Msg("EXIT VariableHandler")
		return err
	}
}

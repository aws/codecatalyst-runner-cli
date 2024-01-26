package actions

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/codecatalyst-runner-cli/command-runner/pkg/runner"

	"github.com/rs/zerolog/log"
)

// ActionInputHandler converts the provided inputs into environment variables (prefixed with 'INPUT_')
func ActionInputHandler(inputs map[string]string) runner.Feature {
	return func(ctx context.Context, plan runner.Plan, e runner.PlanExecutor) error {
		log.Ctx(ctx).Debug().Msg("ENTER ActionInputHandler")
		if inputs == nil {
			log.Ctx(ctx).Debug().Msg("SKIP ActionInputHandler - inputs == nil")
		} else {
			var action *Action
			if ap, ok := plan.(ActionProvider); ok {
				action = ap.Action()
			} else {
				return fmt.Errorf("plan must implement ActionProvider for ActionInputHandler")
			}

			envCfg := plan.EnvironmentConfiguration()
			if envCfg.Env == nil {
				envCfg.Env = make(map[string]string)
			}
			for name, param := range action.Configuration {
				if val, ok := inputs[name]; ok {
					envCfg.Env[fmt.Sprintf("INPUT_%s", strings.ToUpper(name))] = val
				} else if param.Required && param.Default == "" {
					return fmt.Errorf("input parameter '%s' is required for action '%s'", name, action.ID)
				} else {
					envCfg.Env[fmt.Sprintf("INPUT_%s", strings.ToUpper(name))] = param.Default
				}
			}
		}

		err := e(ctx)
		log.Ctx(ctx).Debug().Msg("EXIT ActionInputHandler")
		return err
	}
}

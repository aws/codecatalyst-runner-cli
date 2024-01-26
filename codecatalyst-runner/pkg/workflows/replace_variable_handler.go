package workflows

import (
	"context"
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/aws/codecatalyst-runner-cli/command-runner/pkg/common"
	"github.com/aws/codecatalyst-runner-cli/command-runner/pkg/runner"

	"github.com/rs/zerolog/log"
)

// ReplaceVariableHandler converts the variables to outputs
func ReplaceVariableHandler(planOutputs map[string]map[string]string, secrets SecretProvider) runner.Feature {
	replacer := &variableReplacer{
		planOutputs: planOutputs,
		secrets:     secrets,
	}
	return func(ctx context.Context, plan runner.Plan, e runner.PlanExecutor) error {
		log.Ctx(ctx).Debug().Msg("ENTER ReplaceVariableHandler")
		for name, value := range plan.EnvironmentConfiguration().Env {
			var newValue string
			var err error
			if newValue, err = replacer.Replace(ctx, plan, value); err != nil {
				return err
			}
			plan.EnvironmentConfiguration().Env[name] = newValue
		}
		for _, cg := range plan.CommandGroups() {
			for _, command := range cg.Commands {
				for i, commandPart := range command {
					if newValue, err := replacer.Replace(ctx, plan, commandPart); err != nil {
						return err
					} else {
						command[i] = newValue
					}
				}
			}
		}
		err := e(ctx)
		log.Ctx(ctx).Debug().Msg("EXIT ReplaceVariableHandler")
		return err
	}
}

var replacementVariablePattern = regexp.MustCompile(`\${([a-zA-Z0-9\.\-_]+)\.([a-zA-Z0-9\-_]+)}`)

type variableReplacer struct {
	planOutputs map[string]map[string]string
	secrets     SecretProvider
}

func (v *variableReplacer) Replace(ctx context.Context, plan runner.Plan, value string) (string, error) {
	var rtnError error
	return replacementVariablePattern.ReplaceAllStringFunc(value, func(match string) string {
		varParts := replacementVariablePattern.FindStringSubmatch(match)
		varPrefix := varParts[1]
		varName := varParts[2]
		if varPrefix == "Secrets" {
			if plan.EnvironmentConfiguration().Env == nil {
				plan.EnvironmentConfiguration().Env = make(map[string]string)
			}
			secretEnvName := fmt.Sprintf("CATALYST_SECRETS_%s", varName)
			if secretValue, err := v.secrets.GetSecret(ctx, varName); err != nil {
				log.Ctx(ctx).Err(err).Msg("unable to get secret")
				rtnError = err
			} else {
				plan.EnvironmentConfiguration().Env[secretEnvName] = secretValue
			}
			log.Ctx(ctx).Debug().Msgf("Replacing SECRET %s with %s", varName, secretEnvName)
			return fmt.Sprintf("${%s}", secretEnvName)
		}
		planId := strings.Replace(varPrefix, ".", "@", 1)
		log.Ctx(ctx).Debug().Msgf("Adding DependsOn %s", planId)
		if !slices.Contains(plan.DependsOn(), planId) {
			plan.AddDependsOn(planId)
			rtnError = common.ErrDefer
			return value
		}
		newValue := v.planOutputs[planId][varName]
		log.Ctx(ctx).Debug().Msgf("Replacing VAR %s with %s", match, newValue)
		return newValue
	}), rtnError
}

type SecretProvider interface {
	GetSecret(ctx context.Context, name string) (string, error)
}

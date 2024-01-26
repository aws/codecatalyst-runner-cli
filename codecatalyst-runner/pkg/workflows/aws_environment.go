package workflows

import (
	"context"

	"github.com/aws/codecatalyst-runner-cli/command-runner/pkg/runner"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/rs/zerolog/log"
)

// AWSEnvironment is a feature that will assume a role in AWS.
func AWSEnvironment(profile string) runner.Feature {
	return func(ctx context.Context, plan runner.Plan, e runner.PlanExecutor) error {
		log.Ctx(ctx).Debug().Msg("ENTER AWSEnvironment")
		if plan.EnvironmentConfiguration().Env == nil {
			plan.EnvironmentConfiguration().Env = make(map[string]string)
		}
		env := plan.EnvironmentConfiguration().Env
		cfg, err := config.LoadDefaultConfig(ctx, config.WithSharedConfigProfile(profile))
		if err != nil {
			return err
		}
		credentials, err := cfg.Credentials.Retrieve(ctx)
		if err != nil {
			return err
		}

		env["AWS_ACCESS_KEY_ID"] = credentials.AccessKeyID
		env["AWS_SECRET_ACCESS_KEY"] = credentials.SecretAccessKey
		env["AWS_SESSION_TOKEN"] = credentials.SessionToken
		err = e(ctx)
		log.Ctx(ctx).Debug().Msg("EXIT AWSEnvironment")
		return err
	}
}

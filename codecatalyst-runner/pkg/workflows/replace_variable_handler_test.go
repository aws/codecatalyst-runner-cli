package workflows

import (
	"context"
	"fmt"
	"testing"

	"github.com/aws/codecatalyst-runner-cli/command-runner/pkg/runner"

	"github.com/stretchr/testify/assert"
)

func TestReplaceVariableHandler(t *testing.T) {
	type TestParams struct {
		TestCase          string
		DependsOn         []string
		Env               map[string]string
		Secrets           map[string]string
		Commands          []runner.Command
		PlanOutputs       map[string]map[string]string
		ExpectedEnv       map[string]string
		ExpectedCommands  []runner.Command
		ExpectedDependsOn []string
		ExpectedError     string
	}

	for _, tt := range []*TestParams{
		{
			TestCase: "basic-without-depends-on",
			Env: map[string]string{
				"MyEnv": "hello ${SomeAction.Foo}!",
			},
			ExpectedDependsOn: []string{"SomeAction"},
			ExpectedError:     "deferred",
		},
		{
			TestCase: "basic-with-depends-on",
			Env: map[string]string{
				"MyEnv": "hello ${SomeAction.Foo}!",
			},
			DependsOn: []string{"SomeAction"},
			PlanOutputs: map[string]map[string]string{
				"SomeAction": {
					"Foo": "bar",
				},
			},
			ExpectedDependsOn: []string{"SomeAction"},
			ExpectedEnv: map[string]string{
				"MyEnv": "hello bar!",
			},
		},
		{
			TestCase: "secret",
			Env: map[string]string{
				"MyEnv": "hello ${Secrets.Foo}!",
			},
			Commands: []runner.Command{
				{"echo", "what ${Secrets.Foo}?"},
			},
			Secrets: map[string]string{
				"Foo": "bar",
			},
			ExpectedEnv: map[string]string{
				"MyEnv":                "hello ${CATALYST_SECRETS_Foo}!",
				"CATALYST_SECRETS_Foo": "bar",
			},
			ExpectedCommands: []runner.Command{
				{"echo", "what ${CATALYST_SECRETS_Foo}?"},
			},
		},
		{
			TestCase: "secret-missing",
			Env: map[string]string{
				"MyEnv": "hello ${Secrets.Foo}!",
			},
			Secrets:       map[string]string{},
			ExpectedError: "secret 'Foo' undefined",
		},
	} {
		t.Run(tt.TestCase, func(t *testing.T) {
			assert := assert.New(t)
			// setup the code under test
			ctx := context.Background()
			secrets := newMockSecrets(tt.Secrets)
			feature := ReplaceVariableHandler(tt.PlanOutputs, secrets)

			// setup the mock
			plan := new(runner.MockPlan)
			plan.AddDependsOn(tt.DependsOn...)
			plan.EnvironmentConfiguration().Env = tt.Env
			plan.AddCommandGroup(&runner.CommandGroup{
				Commands: tt.Commands,
			})
			m := new(runner.MockPlanExecutor).WithPlan(plan)
			if tt.ExpectedError == "" {
				m.OnExecute(ctx).Return(nil)
			}

			// run the feature
			err := m.Execute(ctx, feature)

			if tt.ExpectedError != "" {
				assert.Error(err)
				assert.EqualError(err, tt.ExpectedError)
			} else {
				assert.NoError(err)
				assert.Equal(tt.ExpectedEnv, plan.EnvironmentConfiguration().Env)
				assert.Equal(tt.ExpectedCommands, plan.CommandGroups()[0].Commands)
				assert.Equal(tt.ExpectedDependsOn, plan.DependsOn())
			}

			// assert the results
			m.AssertExpectations(t)
		})
	}
}

func newMockSecrets(secrets map[string]string) SecretProvider {
	return &mockSecretProvider{secrets: secrets}
}

type mockSecretProvider struct {
	secrets map[string]string
}

func (m *mockSecretProvider) GetSecret(_ context.Context, secretName string) (string, error) {
	if v, ok := m.secrets[secretName]; ok {
		return v, nil
	}
	return "", fmt.Errorf("secret '%s' undefined", secretName)
}

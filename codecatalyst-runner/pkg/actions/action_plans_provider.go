package actions

import (
	"context"
	"fmt"

	"github.com/aws/codecatalyst-runner-cli/command-runner/pkg/runner"

	"github.com/rs/zerolog/log"
)

// NewActionPlansProviderParams contains the parameters to create a new action plans provider
type NewActionPlansProviderParams struct {
	ActionsProvider Provider             // The actions [Provider] containing the actions to create plans for
	ExecutionType   runner.ExecutionType // The [ExecutionType] to use in the created plans
	WorkingDir      string               // The working directory to use for each plan
}

// NewActionPlansProvider creates a plan provider based on [Action]s
func NewActionPlansProvider(params *NewActionPlansProviderParams) runner.PlansProvider {
	return &actionPlansProvider{
		actionsProvider: params.ActionsProvider,
		executionType:   params.ExecutionType,
		workingDir:      params.WorkingDir,
	}
}

type actionPlansProvider struct {
	actionsProvider Provider
	executionType   runner.ExecutionType
	workingDir      string
}

func (app *actionPlansProvider) Plans(_ context.Context) ([]runner.Plan, error) {
	actions, err := app.actionsProvider.Actions()
	if err != nil {
		return nil, fmt.Errorf("unable to get actions from provider: %w", err)
	}
	plans := make([]runner.Plan, len(actions))
	for i, action := range actions {
		log.Debug().Msgf("creating action plan for action %s", action.ID)
		var plan runner.Plan
		var err error
		if plan, err = NewActionPlan(&NewActionPlanParams{
			Action:        action,
			ExecutionType: app.executionType,
			WorkingDir:    app.workingDir,
		}); err != nil {
			return nil, fmt.Errorf("unable to create new action plan: %w", err)
		}
		plans[i] = plan
	}
	log.Debug().Msgf("created plans=%+v from actions=%+v", plans, actions)
	return plans, nil
}

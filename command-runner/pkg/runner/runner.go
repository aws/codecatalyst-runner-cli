package runner

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"

	"github.com/aws/codecatalyst-runner-cli/command-runner/internal/containers"
	"github.com/aws/codecatalyst-runner-cli/command-runner/pkg/common"

	"github.com/rs/zerolog/log"
)

func newRunner(namespace string, executionType ExecutionType, plan Plan, features ...Feature) common.Executor {
	var executor common.Executor
	executor = func(ctx context.Context) error {
		logPlan("About to execute plan\n", plan)
		for _, commandGroup := range plan.CommandGroups() {
			id := fmt.Sprintf("%s-%s", namespace, plan.ID())
			executor, err := newCommandExecutor(ctx, id, executionType, commandGroup, plan.EnvironmentConfiguration())
			if err != nil {
				return err
			}
			for _, command := range commandGroup.Commands {
				log.Ctx(ctx).Info().Msgf("⚡️ %s", strings.Join(command, " "))
				err := executor.ExecuteCommand(ctx, command)
				if err != nil {
					if closeErr := executor.Close(true); closeErr != nil {
						return errors.Join(err, closeErr)
					}
					return err
				}
			}
			if err := executor.Close(false); err != nil {
				return err
			}
		}
		return nil
	}
	executor = executor.CatchPanic()
	for _, feature := range features {
		executor = newFeatureWrapper(feature, plan).Wrap(executor).CatchPanic()
	}
	return executor
}

func newFeatureWrapper(feature Feature, plan Plan) common.Wrapper {
	return func(ctx context.Context, e common.Executor) error {
		return feature(ctx, plan, PlanExecutor(e))
	}
}

func logPlan(message string, plan Plan) {
	if log.Debug().Enabled() {
		planJSON, _ := json.Marshal(plan)
		log.Debug().Msgf("%s%s", message, planJSON)
	}
}

// RunAllParams contains the input parameters for the RunAll function
type RunAllParams struct {
	Namespace     string           // namespace for this execution
	Plans         PlansProvider    // provider for the list of plans to run
	Features      FeaturesProvider // provider for the features to apply to each plan
	Concurrency   int              // number of plans to run concurrently
	ExecutionType ExecutionType    // executor to use for running commands
}

// PlansProvider returns a list of [Plan]s
type PlansProvider interface {
	Plans(ctx context.Context) ([]Plan, error)
}

// FeaturesProvider returns a list of [Feature]s for a given [Plan]
type FeaturesProvider interface {
	Features(Plan) ([]Feature, error)
}

// RunAll executes all plans and features in parallel
func RunAll(ctx context.Context, params *RunAllParams) error {
	if params.Plans == nil {
		return fmt.Errorf("plannables provider cannot be nil")
	}
	plans, err := params.Plans.Plans(ctx)
	if err != nil {
		return fmt.Errorf("unable to get plans from provider: %w", err)
	}

	executors := make([]common.Executor, 0)
	for _, plan := range plans {
		var features []Feature
		if params.Features != nil {
			features, err = params.Features.Features(plan)
			if err != nil {
				return fmt.Errorf("unable to get features: %w", err)
			}
		}
		executor := newRunner(params.Namespace, params.ExecutionType, plan, features...)
		executors = append(executors, executor)
	}

	concurrency := int(math.Max(1, float64(params.Concurrency)))
	return common.NewParallelExecutor(concurrency, executors...).TraceRegion("actions-runall")(ctx)
}

// ExecutionType allows the caller to force shell or docker execution of the action
type ExecutionType string

const (
	// ExecutionTypeShell configures commands to run in a local shell
	ExecutionTypeShell ExecutionType = "shell"
	// ExecutionTypeDocker configures commands to run in a Docker container
	ExecutionTypeDocker ExecutionType = "docker"
	// ExecutionTypeFinch configures commands to run in a Finch container
	ExecutionTypeFinch ExecutionType = "finch"
)

// DefaultExecutionType determines the appropriate default
func DefaultExecutionType() ExecutionType {
	defaultSP := containers.DefaultServiceProvider(context.Background())
	switch defaultSP {
	case containers.Finch:
		return ExecutionTypeFinch
	case containers.Docker:
		return ExecutionTypeDocker
	default:
		return ExecutionTypeShell
	}
}

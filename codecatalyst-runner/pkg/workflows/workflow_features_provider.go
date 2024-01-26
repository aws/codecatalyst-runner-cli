package workflows

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"github.com/aws/codecatalyst-runner-cli/codecatalyst-runner/pkg/actions"

	"github.com/aws/codecatalyst-runner-cli/command-runner/pkg/features"
	"github.com/aws/codecatalyst-runner-cli/command-runner/pkg/runner"

	"gopkg.in/yaml.v2"
)

// NewWorkflowFeaturesProviderParams contains the params to create a new FeaturesProvider
type NewWorkflowFeaturesProviderParams struct {
	runner.EnvironmentConfiguration                   // The configuration of the environments
	OutputMode                      OutputMode        // Mode to use for output
	NoOutput                        bool              // Disable output from the action execution
	NoCache                         bool              // Disable file caches
	Dryrun                          bool              // Dryrun skips execution of the action
	BindWorkingDir                  bool              // BindWorkingDir will mount the working directory into the container, rather than copying
	EnvironmentProfiles             map[string]string // Map of workflow environment names to AWS CLI profile names
	Workflow                        *Workflow         // Workflow to load features for
	SecretProvider                  SecretProvider    // Secret provider to use for secrets
}

// NewWorkflowFeaturesProvider creates a FeaturesProvider for [Workflow]
func NewWorkflowFeaturesProvider(params *NewWorkflowFeaturesProviderParams) (runner.FeaturesProvider, error) {
	workflowActions := make(map[string]*Action)
	for _, mapItem := range params.Workflow.Actions {
		actionName := mapItem.Key.(string)
		var actionOrGroup ActionOrGroup
		if buf, err := yaml.Marshal(mapItem.Value); err != nil {
			return nil, err
		} else if err := yaml.Unmarshal(buf, &actionOrGroup); err != nil {
			return nil, err
		}
		if actionOrGroup.Action.Identifier != "" {
			workflowActions[actionName] = &actionOrGroup.Action
		} else {
			for subName, action := range actionOrGroup.Actions {
				fullName := fmt.Sprintf("%s@%s", actionName, subName)
				workflowActions[fullName] = action
			}
		}
	}

	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return nil, err
	}
	sha := sha256.Sum256([]byte(params.Workflow.Path))
	workflowHash := hex.EncodeToString(sha[:])
	cacheDir = filepath.Join(cacheDir, "codecatalyst-runner", "workflows", workflowHash)

	secretProvider := params.SecretProvider
	if secretProvider == nil {
		secretProvider = new(envSecretProvider)
	}

	return &workflowFeaturesProvider{
		EnvironmentConfiguration: params.EnvironmentConfiguration,
		cacheDir:                 cacheDir,
		outputMode:               params.OutputMode,
		noOutput:                 params.NoOutput,
		noCache:                  params.NoCache,
		dryrun:                   params.Dryrun,
		bindWorkingDir:           params.BindWorkingDir,
		sharedCompute:            params.Workflow.Compute.SharedInstance,
		workflowActions:          workflowActions,
		artifactPlans:            make(map[string]string),
		environmentProfiles:      params.EnvironmentProfiles,
		planTracker:              new(features.PlanTracker),
		secretProvider:           secretProvider,
	}, nil
}

type OutputMode string

const (
	OutputModeText OutputMode = "text"
	OutputModeTUI  OutputMode = "tui"
)

type workflowFeaturesProvider struct {
	runner.EnvironmentConfiguration
	cacheDir            string
	outputMode          OutputMode
	noOutput            bool
	noCache             bool
	dryrun              bool
	bindWorkingDir      bool
	sharedCompute       bool
	workflowActions     map[string]*Action
	artifactPlans       map[string]string // keep a map with which artifacts depend on which plans
	environmentProfiles map[string]string // map of workflow environment names to AWS CLI profiles
	isWorkingDirSetup   bool
	planTracker         *features.PlanTracker
	secretProvider      SecretProvider
}

var planOutputs = make(map[string]map[string]string)

func (wfp *workflowFeaturesProvider) Features(plan runner.Plan) ([]runner.Feature, error) {
	action := wfp.workflowActions[plan.ID()]

	var outputs map[string]string
	if po, ok := planOutputs[plan.ID()]; ok {
		outputs = po
	} else {
		outputs = make(map[string]string)
		planOutputs[plan.ID()] = outputs
	}

	var loggerFeature runner.Feature
	switch wfp.outputMode {
	case OutputModeTUI:
		loggerFeature = features.TUILogger(plan.ID())
	case OutputModeText:
		loggerFeature = features.ConsoleLogger()
	}
	ft := []runner.Feature{
		features.Reuse(wfp.Reuse),
		actions.ActionOutputHandler(outputs, false),
		features.Dryrun(wfp.dryrun),
	}

	if wfp.sharedCompute || (action != nil && slices.Contains(action.Inputs.Sources, "WorkflowSource")) {
		ft = append(ft,
			features.WorkingDirImporter(wfp.EnvironmentConfiguration.WorkingDir, wfp.cacheDir, wfp.bindWorkingDir, wfp.sharedCompute, &wfp.isWorkingDirSetup),
		)
	}

	if action != nil && action.Environment.Name != "" {
		if profile, ok := wfp.environmentProfiles[action.Environment.Name]; !ok {
			return nil, fmt.Errorf("no AWS profile has been associated with environment %s", action.Environment.Name)
		} else {
			ft = append(ft, AWSEnvironment(profile))
		}
	}

	inputs := make(map[string]string)
	if action != nil {
		for _, input := range action.Inputs.Variables {
			inputs[input.Name] = input.Value
		}
	}

	if action != nil && !wfp.noCache {
		ft = append(ft, FileCache(wfp.EnvironmentConfiguration.WorkingDir, action.Caching.FileCaching, staticCacheDirProvider(wfp.cacheDir)))
	}

	ft = append(ft,
		features.StatusLogger(plan.ID()),
	)
	if action != nil {
		ft = append(ft,
			OutputArtifacts(plan.ID(), action.Outputs.Artifacts, wfp.artifactPlans, wfp.cacheDir),
			InputArtifacts(action.Inputs.Artifacts, wfp.artifactPlans, wfp.cacheDir),
		)
	}
	ft = append(ft,
		ReplaceVariableHandler(planOutputs, wfp.secretProvider),
		InputVariableHandler(inputs),
		features.DependsOn(wfp.planTracker.ProgressHandle(plan.ID())),
		loggerFeature,
	)
	return ft, nil
}

type envSecretProvider struct{}

func (ssp *envSecretProvider) GetSecret(_ context.Context, name string) (string, error) {
	if val, ok := os.LookupEnv(name); ok {
		return val, nil
	}
	return "", fmt.Errorf("secret '%s' undefined", name)
}

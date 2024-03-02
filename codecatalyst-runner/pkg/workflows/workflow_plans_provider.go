package workflows

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/aws/codecatalyst-runner-cli/codecatalyst-runner/pkg/actions"

	"github.com/aws/codecatalyst-runner-cli/command-runner/pkg/runner"

	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v2"
)

var ActionsUrlTemplate = "https://amazon-codecatalyst-public-action-source-us-west-2.s3.us-west-2.amazonaws.com/us-west-2/%s/%s/action-repo.zip"
var ActionVersions = map[string]string{
	"aws/kubernetes-deploy":          "1.0.0",
	"aws/ecs-render-task-definition": "1.0.4",
	"aws/cfn-deploy":                 "1.0.5",
	"aws/ecs-deploy":                 "1.0.5",
	"aws/cdk-deploy":                 "1.0.13",
	"aws/cdk-bootstrap":              "1.0.8",
	"aws/s3-publish":                 "1.0.5",
	"aws/lambda-invoke":              "1.0.8",
	"codecatalyst-labs/provision-with-terraform-community": "1.0.0",
	"codecatalyst-labs/scan-with-codeguru-security":        "1.0.0",
	"codecatalyst-labs/deploy-to-cloudfront-s3":            "1.0.1",
	"codecatalyst-labs/publish-to-codeartifact":            "1.0.1",
	"codecatalyst-labs/invalidate-cloudfront-cache":        "1.0.0",
	"codecatalyst-labs/publish-to-sns":                     "1.0.0",
	"codecatalyst-labs/deploy-to-app-runner":               "1.0.3",
	"codecatalyst-labs/outgoing-webhook":                   "1.0.1",
	"codecatalyst-labs/deploy-with-sam":                    "1.0.1",
	"codecatalyst-labs/push-to-ecr":                        "1.0.3",
	"codecatalyst-labs/deploy-to-amplify-hosting":          "1.0.1",
	"mend/mendsca": "1.0.9",
}

// NewWorkflowPlansProviderParams contains the parameters to create a new action plans provider
type NewWorkflowPlansProviderParams struct {
	ExecutionType runner.ExecutionType // The [ExecutionType] to use in the created plans
	WorkingDir    string               // The working directory to use for each plan
	Action        string               // the name of the action to run
	Workflow      *Workflow            // The [Workflow] to use
}

// NewWorkflowPlansProvider creates a plan provider based on [Workflow]s
func NewWorkflowPlansProvider(params *NewWorkflowPlansProviderParams) runner.PlansProvider {
	return &workflowPlansProvider{
		executionType: params.ExecutionType,
		workingDir:    params.WorkingDir,
		action:        params.Action,
		workflow:      params.Workflow,
	}
}

type workflowPlansProvider struct {
	executionType runner.ExecutionType
	workingDir    string
	action        string
	workflow      *Workflow
}

func (wpp *workflowPlansProvider) Plans(ctx context.Context) ([]runner.Plan, error) {
	plans := make([]runner.Plan, 0)
	for _, mapItem := range wpp.workflow.Actions {
		actionName := mapItem.Key.(string)
		var actionOrGroup ActionOrGroup
		if buf, err := yaml.Marshal(mapItem.Value); err != nil {
			return nil, err
		} else if err := yaml.Unmarshal(buf, &actionOrGroup); err != nil {
			return nil, err
		}
		if actionOrGroup.Action.Identifier != "" {
			plan, err := wpp.planAction(ctx, actionName, &actionOrGroup.Action)
			if err != nil {
				return nil, fmt.Errorf("unable to create plan for action %s: %w", actionName, err)
			}
			if plan != nil {
				plans = append(plans, plan)
			}
		} else {
			for subName, action := range actionOrGroup.Actions {
				fullName := fmt.Sprintf("%s@%s", actionName, subName)
				plan, err := wpp.planAction(ctx, fullName, action)
				if err != nil {
					return nil, fmt.Errorf("unable to create plan for action %s: %w", fullName, err)
				}
				if plan != nil {
					plans = append(plans, plan)
				}
			}
		}
	}
	if log.Debug().Enabled() {
		log.Debug().Msgf("created plans from workflow=%+v", wpp.workflow)
		for _, plan := range plans {
			log.Debug().Msgf("  plan=%+v", plan)
		}
	}
	return plans, nil
}

func (wpp *workflowPlansProvider) planAction(ctx context.Context, actionName string, action *Action) (runner.Plan, error) {
	if wpp.action != "" && wpp.action != actionName {
		return nil, nil
	}
	log.Ctx(ctx).Debug().Msgf("creating action plan for action %s", action.Identifier)
	var plan runner.Plan
	var err error
	var actionSpec *actions.Action
	var steps []string
	actionIdentifierParts := strings.Split(action.Identifier, "@")
	switch actionIdentifierParts[0] {
	case ".":
		actionSpec, err = actions.Load(wpp.workingDir)
		if err != nil {
			return nil, fmt.Errorf("unable to load action file '%s': %w", wpp.workingDir, err)
		}
	case "aws/build", "aws/managed-test":
		var runs actions.Runs
		runs = actions.Runs{
			Using:      actions.UsingTypeDocker,
			Image:      actions.CodeCatalystImage(),
			Entrypoint: "/bin/echo",
		}
		outputs := actions.Outputs{
			Variables: make(map[string]actions.Output),
		}
		for _, output := range action.Outputs.Variables {
			outputs.Variables[output] = actions.Output{}
		}
		actionSpec = &actions.Action{
			SchemaVersion: "1.0",
			ID:            actionIdentifierParts[0],
			Name:          actionIdentifierParts[0],
			Version:       actionIdentifierParts[1],
			Runs:          runs,
			Outputs:       outputs,
		}
		steps = make([]string, 0)
		if configSteps, ok := action.Configuration["Steps"].([]interface{}); ok {
			for _, step := range configSteps {
				steps = append(steps, step.(map[interface{}]interface{})["Run"].(string))
			}
		}
	case "aws/github-actions-runner":
		return nil, fmt.Errorf("GitHub actions are not currently supported")
	default:
		actionSpec, err = loadRemoteAction(ctx, actionIdentifierParts[0])
		if err != nil {
			return nil, err
		}
	}

	log.Ctx(ctx).Debug().Msgf("actionspec=%+v", actionSpec)
	if plan, err = actions.NewActionPlan(&actions.NewActionPlanParams{
		Action:        actionSpec,
		ExecutionType: wpp.executionType,
		WorkingDir:    wpp.workingDir,
		ID:            actionName,
		Steps:         steps,
		DependsOn:     action.DependsOn,
	}); err != nil {
		return nil, fmt.Errorf("unable to create new action plan: %w", err)
	}
	err = applyInputs(plan.EnvironmentConfiguration(), action, actionSpec)
	return plan, err
}

func applyInputs(envCfg *runner.EnvironmentConfiguration, action *Action, actionSpec *actions.Action) error {
	if envCfg.Env == nil {
		envCfg.Env = make(map[string]string)
	}
	for name, param := range actionSpec.Configuration {
		if val, ok := action.Configuration[name]; ok {
			envCfg.Env[fmt.Sprintf("INPUT_%s", strings.ToUpper(name))] = val.(string)
		} else if param.Required && param.Default == "" {
			return fmt.Errorf("input parameter '%s' is required for action '%s'", name, actionSpec.ID)
		} else {
			envCfg.Env[fmt.Sprintf("INPUT_%s", strings.ToUpper(name))] = param.Default
		}
	}
	return nil
}

func downloadHttpExtractZip(_ context.Context, url string, destDir string) error {
	response, err := http.Get(url) // #nosec G107 -- URLs are generated above from trusted host
	if err != nil {
		return fmt.Errorf("unable to get object from url %s: %w", url, err)
	}
	defer response.Body.Close()

	actionZip, err := os.CreateTemp("", "actions-*.zip")
	if err != nil {
		return fmt.Errorf("unable to create temp file: %w", err)
	}
	defer os.Remove(actionZip.Name())
	if _, err = io.Copy(actionZip, response.Body); err != nil {
		return fmt.Errorf("unable to copy zip to temp file: %w", err)
	}
	_ = actionZip.Close()

	archive, err := zip.OpenReader(actionZip.Name())
	if err != nil {
		return fmt.Errorf("unable to open zip: %w", err)
	}
	defer archive.Close()
	for _, f := range archive.File {
		filePath := filepath.Join(destDir, f.Name) //#nosec G305 -- mitigated through next line
		if !strings.HasPrefix(filePath, filepath.Clean(destDir)+string(os.PathSeparator)) {
			return fmt.Errorf("invalid file path: %s", filePath)
		}
		if f.FileInfo().IsDir() {
			err = os.MkdirAll(filePath, os.ModePerm)
			if err != nil {
				return fmt.Errorf("unable to create directory: %w", err)
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(filePath), os.ModePerm); err != nil {
			return fmt.Errorf("unable to create directory: %w", err)
		}

		dstFile, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return fmt.Errorf("unable to open file: %w", err)
		}

		fileInArchive, err := f.Open()
		if err != nil {
			return fmt.Errorf("unable to open file in archive: %w", err)
		}

		for {
			_, err := io.CopyN(dstFile, fileInArchive, 1024)
			if err != nil {
				if err == io.EOF {
					break
				}
				return fmt.Errorf("unable to copy file: %w", err)
			}
		}

		dstFile.Close()
		fileInArchive.Close()
	}
	return nil
}

func loadRemoteAction(ctx context.Context, actionID string) (*actions.Action, error) {
	if actionVersion, ok := ActionVersions[actionID]; !ok {
		return nil, fmt.Errorf("unknown actions %s", actionID)
	} else {
		actionsURL := fmt.Sprintf(ActionsUrlTemplate, actionID, actionVersion)
		log.Ctx(ctx).Info().Msgf("🚚 downloading action %s", actionID)
		sha := sha256.Sum256([]byte(actionsURL))
		actionsURLHash := hex.EncodeToString(sha[:])
		cacheDir, err := os.UserCacheDir()
		if err != nil {
			return nil, err
		}
		actionDir := filepath.Join(cacheDir, "codecatalyst-runner", "actions", actionsURLHash)
		if err := os.RemoveAll(actionDir); err != nil {
			return nil, fmt.Errorf("unable to cleanup actionDir %s: %w", actionDir, err)
		}
		if err := downloadHttpExtractZip(ctx, actionsURL, actionDir); err != nil {
			return nil, fmt.Errorf("unable to download actionsUrl %s: %w", actionsURL, err)
		}
		if entries, err := os.ReadDir(actionDir); err != nil {
			return nil, fmt.Errorf("unable to list actionDir %s: %w", actionDir, err)
		} else {
			for _, entry := range entries {
				if entry.IsDir() && strings.HasPrefix(entry.Name(), "cloned-repo-") {
					actionDir = filepath.Join(actionDir, entry.Name())
				}
			}
		}
		return actions.Load(actionDir)
	}
}

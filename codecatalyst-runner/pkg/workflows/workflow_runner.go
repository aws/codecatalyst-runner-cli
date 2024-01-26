package workflows

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"

	"github.com/aws/codecatalyst-runner-cli/command-runner/pkg/runner"

	"github.com/manifoldco/promptui"
	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v2"
)

type RunParams struct {
	NewWorkflowPlansProviderParams
	NewWorkflowFeaturesProviderParams
	Concurrency  int
	WorkflowPath string
	WorkflowName string
}

func Run(ctx context.Context, params *RunParams) error {
	log.Ctx(ctx).Debug().Msgf("running workflow with params %+v", *params)
	if params.WorkflowPath != "" {
		if _, err := os.Stat(params.WorkflowPath); err != nil {
			return fmt.Errorf("unable to load workflow file '%s': %w", params.WorkflowPath, err)
		}

		params.WorkingDir, _ = filepath.Abs(filepath.Dir(filepath.Dir(filepath.Dir(params.WorkflowPath))))
	} else {
		params.WorkingDir, _ = filepath.Abs(params.WorkingDir)
		if workflows, err := os.ReadDir(filepath.Join(params.WorkingDir, ".codecatalyst", "workflows")); err != nil {
			return err
		} else {
			workflowOptions := make(map[string]string, 0)
			for _, workflow := range workflows {
				ext := filepath.Ext(workflow.Name())
				if ext != ".yml" && ext != ".yaml" {
					continue
				}
				workflowFile := filepath.Join(params.WorkingDir, ".codecatalyst", "workflows", workflow.Name())
				log.Debug().Msgf("considering workflow file %s", workflowFile)
				if workflow, err := readWorkflow(workflowFile); err != nil {
					return fmt.Errorf("unable to read workflow file '%s': %w", workflowFile, err)
				} else {
					workflowOptions[workflow.Name] = workflowFile
				}
			}
			if params.WorkflowName != "" {
				if val, ok := workflowOptions[params.WorkflowName]; !ok {
					return fmt.Errorf("no workflow defined named '%s'", params.WorkflowName)
				} else {
					params.WorkflowPath = val
				}
			} else {
				// prompt to select a workflow
				prompt := promptui.Select{
					Label:  "Select workflow",
					Items:  reflect.ValueOf(workflowOptions).MapKeys(),
					Stdout: &bellSkipper{},
				}
				if _, result, err := prompt.Run(); err != nil {
					return fmt.Errorf("unable to select a workflow: %w", err)
				} else {
					params.WorkflowPath = workflowOptions[result]
				}
			}
		}
	}

	if !filepath.IsAbs(params.WorkflowPath) {
		if absWorkflowPath, err := filepath.Abs(params.WorkflowPath); err != nil {
			return err
		} else {
			params.WorkflowPath = absWorkflowPath
		}
	}

	log.Debug().Msgf("🚚 Running workflow file '%s'", params.WorkflowPath)

	workflow, err := readWorkflow(params.WorkflowPath)
	if err != nil {
		return fmt.Errorf("unable to read workflow file '%s': %w", params.WorkflowPath, err)
	}

	params.NewWorkflowPlansProviderParams.Workflow = workflow
	plans := NewWorkflowPlansProvider(&params.NewWorkflowPlansProviderParams)

	params.NewWorkflowFeaturesProviderParams.Workflow = workflow
	params.NewWorkflowFeaturesProviderParams.EnvironmentConfiguration.WorkingDir = params.WorkingDir
	features, err := NewWorkflowFeaturesProvider(&params.NewWorkflowFeaturesProviderParams)
	if err != nil {
		return fmt.Errorf("unable to create features provider: %w", err)
	}
	return runner.RunAll(ctx, &runner.RunAllParams{
		Namespace:     workflow.Name,
		Plans:         plans,
		Features:      features,
		Concurrency:   params.Concurrency,
		ExecutionType: params.ExecutionType,
	})
}

func readWorkflow(workflowPath string) (*Workflow, error) {
	workflow := &Workflow{
		Path: workflowPath,
	}
	if workflowContent, err := os.ReadFile(workflowPath); err != nil {
		return nil, fmt.Errorf("unable to read workflow file '%s': %w", workflowPath, err)
	} else if err = yaml.Unmarshal(workflowContent, workflow); err != nil {
		return nil, fmt.Errorf("unable to unmarshal workflow file '%s': %w", workflowPath, err)
	} else if workflow.SchemaVersion != "1.0" {
		return nil, fmt.Errorf("unsupported SchemaVersion=%s found in workflow %s", workflow.SchemaVersion, workflowPath)
	}
	return workflow, nil
}

// bellSkipper implements an io.WriteCloser that skips the terminal bell
// character (ASCII code 7), and writes the rest to os.Stderr. It is used to
// replace readline.Stdout, that is the package used by promptui to display the
// prompts.
//
// This is a workaround for the bell issue documented in
// https://github.com/manifoldco/promptui/issues/49.
type bellSkipper struct{}

// Write implements an io.WriterCloser over os.Stderr, but it skips the terminal
// bell character.
func (bs *bellSkipper) Write(b []byte) (int, error) {
	const charBell = 7 // c.f. readline.CharBell
	if len(b) == 1 && b[0] == charBell {
		return 0, nil
	}
	return os.Stderr.Write(b)
}

// Close implements an io.WriterCloser over os.Stderr.
func (bs *bellSkipper) Close() error {
	return os.Stderr.Close()
}

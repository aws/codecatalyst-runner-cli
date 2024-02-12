package actions

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/aws/codecatalyst-runner-cli/command-runner/pkg/runner"

	"github.com/kballard/go-shellquote"
	"github.com/rs/zerolog/log"
)

// NewActionPlanParams contains the parametes for the NewActionPlan function
type NewActionPlanParams struct {
	Action        *Action              // Action to use to create a plan from
	ExecutionType runner.ExecutionType // Type of execution (shell or docker)
	WorkingDir    string               // Working directory for the plan
	ID            string               // override the id of the action
	Steps         []string             // override commands to run
	DependsOn     []string             // dependencies
}

const CodeCatalystImage = "docker://public.ecr.aws/c8t2t1h8/al2/curated:1.3-x86_64-ec2"
const containerActionDir = "/codecatalyst/output/action"

// NewActionPlan creates a new Plan from the given params
func NewActionPlan(params *NewActionPlanParams) (runner.Plan, error) {
	var id string
	if params.ID != "" {
		id = params.ID
	} else {
		id = params.Action.ID
	}
	workingDir, err := filepath.Abs(params.WorkingDir)
	if err != nil {
		return nil, err
	}
	actionPlan := &actionPlan{
		id:            id,
		action:        params.Action,
		commandGroups: make([]*runner.CommandGroup, 0),
		environmentConfiguration: &runner.EnvironmentConfiguration{
			FileMaps: make([]*runner.FileMap, 0),
			Env: map[string]string{
				"CATALYST_WORKFLOW_SPACE_NAME":   "-",
				"CATALYST_WORKFLOW_SPACE_ID":     "-",
				"CATALYST_WORKFLOW_PROJECT_NAME": "-",
				"CATALYST_WORKFLOW_PROJECT_ID":   "-",
				"CI":                             "true",
			},
			WorkingDir: workingDir,
		},
		dependsOn:     params.DependsOn,
		ExecutionType: params.ExecutionType,
	}

	if params.Action.Runs.Using == UsingTypeDocker {
		err = actionPlan.loadDockerAction(params.Action, params.Steps)
	} else {
		err = actionPlan.loadNodeAction(params.Action, params.Steps, params.ExecutionType)
	}
	return actionPlan, err
}

type actionPlan struct {
	id                       string
	dependsOn                []string
	environmentConfiguration *runner.EnvironmentConfiguration
	commandGroups            []*runner.CommandGroup
	action                   *Action
	ExecutionType            runner.ExecutionType
}

func (ap *actionPlan) EnvironmentConfiguration() *runner.EnvironmentConfiguration {
	return ap.environmentConfiguration
}
func (ap *actionPlan) CommandGroups() []*runner.CommandGroup {
	return ap.commandGroups
}

func (ap *actionPlan) Action() *Action {
	return ap.action
}

func (ap *actionPlan) ID() string {
	return ap.id
}

func (ap *actionPlan) DependsOn() []string {
	return ap.dependsOn
}

func (ap *actionPlan) AddDependsOn(dependencies ...string) {
	ap.dependsOn = append(ap.dependsOn, dependencies...)
}

func newCommandGroup(image string, entrypoint string) (*runner.CommandGroup, error) {
	entrypointParts, err := shellquote.Split(entrypoint)
	if err != nil {
		return nil, err
	}
	return &runner.CommandGroup{
		Image:      image,
		Entrypoint: entrypointParts,
	}, nil
}

// ActionProvider exposes acces to an [Action]
type ActionProvider interface {
	Action() *Action
}

func (ap *actionPlan) loadDockerAction(action *Action, steps []string) error {
	image := action.Runs.Image
	if !strings.HasPrefix(image, "docker://") {
		image = filepath.Join(action.Basedir, image)
	}
	for i, cmd := range []string{action.Runs.PreEntryPoint, action.Runs.Entrypoint, action.Runs.PostEntryPoint} {
		if cmd != "" {
			entrypoint := "/bin/cat"
			cg, err := newCommandGroup(image, entrypoint)
			if err != nil {
				return err
			}
			cg.Commands = append(cg.Commands, []string{cmd})
			if i == 1 { // add steps to the main command group
				log.Debug().Msgf("steps: %+v", steps)
				for _, step := range steps {
					if step != "" {
						cg.Commands = append(cg.Commands, []string{step})
					}
				}
			}
			log.Debug().Msgf("adding command group: %+v", cg)
			ap.commandGroups = append(ap.commandGroups, cg)
		}
	}
	return nil
}

func (ap *actionPlan) loadNodeAction(action *Action, steps []string, executionType runner.ExecutionType) error {
	var image, entrypoint string
	if executionType == runner.ExecutionTypeDocker || executionType == runner.ExecutionTypeFinch {
		switch action.Runs.Using {
		case UsingTypeNode12:
			image = CodeCatalystImage
		case UsingTypeNode16:
			image = CodeCatalystImage
		default:
			return fmt.Errorf("unsupported value for 'using': %s", action.Runs.Using)
		}
		entrypoint = "/bin/cat"
		ap.environmentConfiguration.FileMaps = append(ap.environmentConfiguration.FileMaps, &runner.FileMap{
			SourcePath: action.Basedir,
			TargetPath: containerActionDir,
			Type:       runner.FileMapTypeCopyInWithGitignore,
		})
	}
	for i, command := range []string{action.Runs.Pre, action.Runs.Main, action.Runs.Post} {
		if command != "" {
			var fullCommand string
			switch executionType {
			case runner.ExecutionTypeDocker, runner.ExecutionTypeFinch:
				fullCommand = filepath.Join(containerActionDir, action.ID, command)
				ap.environmentConfiguration.Env["CATALYST_SOURCE_DIR_CawsCustomActionSource"] = containerActionDir
			case runner.ExecutionTypeShell:
				var err error
				fullCommand, err = filepath.Abs(fmt.Sprintf("%s/%s", action.Basedir, command))
				ap.environmentConfiguration.Env["CATALYST_SOURCE_DIR_CawsCustomActionSource"] = action.Basedir
				if err != nil {
					return err
				}
			default:
				return fmt.Errorf("unsupported execution type: %s", executionType)
			}
			cg, err := newCommandGroup(image, entrypoint)
			if err != nil {
				return err
			}
			cg.Commands = append(cg.Commands, []string{"node", fullCommand})
			if i == 1 { // add steps to the main command group
				for _, step := range steps {
					cg.Commands = append(cg.Commands, []string{step})
				}
			}
			ap.commandGroups = append(ap.commandGroups, cg)
		}
	}
	return nil
}

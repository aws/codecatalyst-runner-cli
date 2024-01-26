package actions

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v2"
)

// Action defines inputs, outputs and resources integrations for custom actions with. For more details,
// see the [Action Reference] in the Amazon CodeCatalyst Action Developer Kit (ADK).
//
// [Action Reference]: https://docs.aws.amazon.com/codecatalyst/latest/adk/action-ref.html
type Action struct {
	SchemaVersion        string               `yaml:"SchemaVersion"`        // version of the action schema
	Name                 string               `yaml:"Name"`                 // name of the action
	ID                   string               `yaml:"Id"`                   // ID of the action
	Description          string               `yaml:"Description"`          // description of the action
	Version              string               `yaml:"Version"`              // version of the action
	Configuration        map[string]Parameter `yaml:"Configuration"`        // define the configuration properties of the action
	SupportedComputeType SupportedComputeType `yaml:"SupportedComputeType"` // specify the compute types to use for the action
	Environment          Environment          `yaml:"Environment"`          // specify the CodeCatalyst environment to use with the action
	Inputs               Inputs               `yaml:"Inputs"`               // defines the data that an action needs during a workflow run
	Outputs              Outputs              `yaml:"Outputs"`              // defines the data that is output by the action during a workflow run. If more than 10 output variables are produced, the top 10 variables are selected
	Runs                 Runs                 `yaml:"Runs"`                 // defines the runtime environment and main entry point for the action
	Basedir              string               `yaml:"-"`                    // the directory this action was loaded from
}

// SupportedComputeType is the compute type to use for the action. You can specify the following types: EC2, Lambda
type SupportedComputeType string

const (
	// SupportedComputeTypeEc2 is EC2 compute type
	SupportedComputeTypeEc2 SupportedComputeType = "EC2"
	// SupportedComputeTypeLambda is Lambda compute type
	SupportedComputeTypeLambda SupportedComputeType = "LAMBDA"
)

// Load an action by reading the '.codecatalyst/actions/action.yml file relative to the provided actionDir
func Load(actionDir string) (*Action, error) {
	action := new(Action)
	actionSpecPath := filepath.Join(actionDir, ".codecatalyst", "actions", "action.yml")
	actionFile, err := os.Open(actionSpecPath)
	if err != nil {
		return nil, err
	}
	defer actionFile.Close()
	err = yaml.NewDecoder(actionFile).Decode(&action)
	action.Basedir = actionDir
	if err != nil {
		return nil, fmt.Errorf("unable to parse %s: %s", actionSpecPath, err.Error())
	}
	if action.ID == "" {
		action.ID = filepath.Base(actionDir)
	}
	action.ID = strings.ReplaceAll(action.ID, "/", "")
	return action, nil
}

package actions

// Parameter configuration in the action
type Parameter struct {
	Description string        `yaml:"Description"` // provide a description of the parameter
	Required    bool          `yaml:"Required"`    // specify whether the parameter is required
	Default     string        `yaml:"Default"`     // specify the default value of the parameter
	DisplayName string        `yaml:"DisplayName"` // set the display name of the parameter
	Type        ParameterType `yaml:"Type"`        // the type of the parameter: number, boolean, or string
}

// ParameterType of parameter. You can use one of the following values (default is string): number, boolean, or string
type ParameterType string

const (
	// ParameterTypeNumber is a number
	ParameterTypeNumber ParameterType = "number"
	// ParameterTypeBoolean is a boolean
	ParameterTypeBoolean ParameterType = "boolean"
	// ParameterTypeString is a string
	ParameterTypeString ParameterType = "string"
)

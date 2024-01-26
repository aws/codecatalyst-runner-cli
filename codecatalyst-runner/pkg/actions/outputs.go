package actions

// Outputs defines the data that is output by the action during a workflow run.
// If more than 10 output variables are produced, the top 10 variables are selected.
type Outputs struct {
	Variables map[string]Output `yaml:"Variables"` // specify the variables that you want the action to export so that they are available for use by the subsequent actions.
}

// Output variables that you want the action to export so that they are available for use by the subsequent actions.
type Output struct {
	Description string `yaml:"Description"` // provide a description of the output variable
}

package actions

// Inputs specifies if sources and/or artifacts are required for the action to run
type Inputs struct {
	Sources   Sources   // Specify the labels that represent the source repositories that will be needed by the action
	Artifacts Artifacts // Specify artifacts from previous actions that you want to provide as input to this action
}

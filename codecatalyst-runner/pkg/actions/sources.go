package actions

// Sources specifies the labels that represent the source repositories that will be needed by the action.
// Currently, the only supported label is WorkflowSource, which represents the source repository
// where your workflow definition file is stored.
type Sources struct {
	Required bool `yaml:"Required"` // if true, then action expects at least one source
}

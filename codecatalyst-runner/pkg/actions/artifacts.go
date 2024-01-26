package actions

// Artifacts from previous actions that you want to provide as input to this action.
// These artifacts must already be defined as output artifacts in previous actions.
type Artifacts struct {
	Required bool `yaml:"Required"` // if true, then action expects at least one artifact
}

package actions

// Environment to use with the Action
type Environment struct {
	Required bool `yaml:"Required"` // if true, the action requires an environment
}

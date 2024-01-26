package actions

// Provider describes an interface for providing a list of [Action]s
type Provider interface {
	Actions() ([]*Action, error)
}

type staticActionsProvider struct {
	actions []*Action
}

// Actions returns the list of [Action]s
func (sap *staticActionsProvider) Actions() ([]*Action, error) {
	return sap.actions, nil
}

// NewStaticActionsProvider creates a new [Provider] for a static list of actions
func NewStaticActionsProvider(actions ...*Action) Provider {
	return &staticActionsProvider{
		actions: actions,
	}
}

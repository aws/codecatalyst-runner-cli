package actions

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStaticActionsProvider(t *testing.T) {
	assert := assert.New(t)

	// setup
	actions := []*Action{
		{
			ID: "test1",
		},
		{
			ID: "test2",
		},
	}
	actionsProvider := NewStaticActionsProvider(actions...)

	actualActions, err := actionsProvider.Actions()
	assert.NoError(err)
	assert.Equal(actions, actualActions)
}

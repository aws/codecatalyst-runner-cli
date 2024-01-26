package actions

import (
	"context"
	"testing"

	"github.com/aws/codecatalyst-runner-cli/command-runner/pkg/runner"

	"github.com/stretchr/testify/assert"
)

func TestNewActionPlansProvider(t *testing.T) {
	assert := assert.New(t)

	// setup
	actions := []*Action{
		{ID: "test1"},
		{ID: "test2"},
	}
	actionsProvider := NewStaticActionsProvider(actions...)
	plansProvider := NewActionPlansProvider(&NewActionPlansProviderParams{
		ActionsProvider: actionsProvider,
		ExecutionType:   runner.ExecutionTypeShell,
		WorkingDir:      ".",
	})

	plans, err := plansProvider.Plans(context.Background())
	assert.NoError(err)
	assert.Len(plans, 2)
	assert.Equal("test1", plans[0].ID(), "test1 ID")
	assert.Equal("test2", plans[1].ID(), "test1 ID")
}

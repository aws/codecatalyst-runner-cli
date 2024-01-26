package actions

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFind(t *testing.T) {
	assert := assert.New(t)

	type TestParams struct {
		TestCase          string
		SearchPath        string
		ExpectedActionIDs []string
	}

	for _, tt := range []*TestParams{
		{
			TestCase:          "Basic",
			SearchPath:        "testdata/actions/two-actions",
			ExpectedActionIDs: []string{"test1", "test2"},
		},
		{
			TestCase:          "Direct",
			SearchPath:        "testdata/actions/two-actions/test1",
			ExpectedActionIDs: []string{"test1"},
		},
		{
			TestCase:          "Empty",
			SearchPath:        "testdata/actions/empty",
			ExpectedActionIDs: []string{},
		},
	} {
		// setup the code under test
		actions, err := Find(tt.SearchPath)
		assert.NoError(err)
		assert.Len(actions, len(tt.ExpectedActionIDs), "%s - Actions len", tt.TestCase)
		for i, actionID := range tt.ExpectedActionIDs {
			assert.Equal(actionID, actions[i].ID, "%s - Action ID %d", tt.TestCase, i)
		}
	}
}

package actions

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoad(t *testing.T) {
	assert := assert.New(t)

	type TestParams struct {
		TestCase         string
		ActionDir        string
		ExpectedActionID string
		ExpectedErr      string
	}

	for _, tt := range []*TestParams{
		{
			TestCase:         "Basic",
			ActionDir:        "testdata/actions/two-actions/test1",
			ExpectedActionID: "test1",
		},
		{
			TestCase:    "Empty",
			ActionDir:   "testdata/actions/empty",
			ExpectedErr: "open testdata/actions/empty/.codecatalyst/actions/action.yml: no such file or directory",
		},
	} {
		// setup the code under test
		action, err := Load(tt.ActionDir)
		if tt.ExpectedActionID != "" {
			assert.NoError(err)
			assert.Equal(tt.ExpectedActionID, action.ID, "%s - Action ID", tt.TestCase)
		}
		if tt.ExpectedErr != "" {
			assert.EqualError(err, tt.ExpectedErr, "%s - Error", tt.TestCase)
		}
	}
}

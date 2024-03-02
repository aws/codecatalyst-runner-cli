package runner

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExecutor(t *testing.T) {
	assert := assert.New(t)

	type TestParams struct {
		TestCase      string
		CommandGroup  CommandGroup
		ExecutionType ExecutionType
		ExpectedType  interface{}
	}

	for _, tt := range []*TestParams{
		{
			TestCase:      "Shell",
			CommandGroup:  CommandGroup{},
			ExecutionType: ExecutionTypeShell,
			ExpectedType:  &shellCommandExecutor{},
		},
		{
			TestCase: "Container",
			CommandGroup: CommandGroup{
				Image:      "docker://alpine:latest",
				Entrypoint: []string{"/bin/ls"},
			},
			ExecutionType: ExecutionTypeDocker,
			ExpectedType:  &containerCommandExecutor{},
		},
	} {
		// setup the code under test
		ctx := context.Background()
		executor, err := newCommandExecutor(ctx, "testID", tt.ExecutionType, &tt.CommandGroup, &EnvironmentConfiguration{
			Env: map[string]string{
				"CATALYST_SOURCE_DIR_WorkflowSource": "foo",
			},
		})
		assert.NoError(err, "%s - newCommandExecutor()", tt.TestCase)
		defer executor.Close(false)

		assert.IsType(tt.ExpectedType, executor, "%s - type", tt.TestCase)
	}
}

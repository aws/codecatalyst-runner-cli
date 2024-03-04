package workflows

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/aws/codecatalyst-runner-cli/command-runner/pkg/runner"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
)

func TestRun(t *testing.T) {
	ctx := context.Background()

	if testing.Short() {
		t.Skip("skipping test in short mode")
	}

	type TestRunTargetParams struct {
		TestCase      string
		WorkflowPath  string
		Secrets       map[string]string
		ExecutionType runner.ExecutionType
		ExpectError   error
	}

	for _, tt := range []*TestRunTargetParams{
		{
			TestCase:      "sample-finch",
			WorkflowPath:  "testdata/exemplar-codecatalyst-action/.codecatalyst/workflows/sample.yaml",
			ExecutionType: runner.ExecutionTypeFinch,
			Secrets: map[string]string{
				"SAMPLE_SECRET": "mysecretvalue",
			},
		},
		{
			TestCase:      "sample-docker",
			WorkflowPath:  "testdata/exemplar-codecatalyst-action/.codecatalyst/workflows/sample.yaml",
			ExecutionType: runner.ExecutionTypeDocker,
			Secrets: map[string]string{
				"SAMPLE_SECRET": "mysecretvalue",
			},
		},
		{
			TestCase:      "sample-shell",
			WorkflowPath:  "testdata/exemplar-codecatalyst-action/.codecatalyst/workflows/sample.yaml",
			ExecutionType: runner.ExecutionTypeShell,
			Secrets: map[string]string{
				"SAMPLE_SECRET": "mysecretvalue",
			},
		},
		{
			TestCase:      "cache-finch",
			WorkflowPath:  "testdata/exemplar-codecatalyst-action/.codecatalyst/workflows/cache.yaml",
			ExecutionType: runner.ExecutionTypeFinch,
		},
		{
			TestCase:      "cache-docker",
			WorkflowPath:  "testdata/exemplar-codecatalyst-action/.codecatalyst/workflows/cache.yaml",
			ExecutionType: runner.ExecutionTypeDocker,
		},
		{
			TestCase:      "cache-shell",
			WorkflowPath:  "testdata/exemplar-codecatalyst-action/.codecatalyst/workflows/cache-shell.yaml",
			ExecutionType: runner.ExecutionTypeShell,
		},
		{
			TestCase:      "custom-finch",
			WorkflowPath:  "testdata/exemplar-codecatalyst-action/.codecatalyst/workflows/custom.yaml",
			ExecutionType: runner.ExecutionTypeFinch,
		},
		{
			TestCase:      "custom-docker",
			WorkflowPath:  "testdata/exemplar-codecatalyst-action/.codecatalyst/workflows/custom.yaml",
			ExecutionType: runner.ExecutionTypeDocker,
		},
		{
			TestCase:      "custom-shell",
			WorkflowPath:  "testdata/exemplar-codecatalyst-action/.codecatalyst/workflows/custom.yaml",
			ExecutionType: runner.ExecutionTypeShell,
		},
		{
			TestCase:      "shared-finch",
			WorkflowPath:  "testdata/exemplar-codecatalyst-action/.codecatalyst/workflows/shared.yaml",
			ExecutionType: runner.ExecutionTypeFinch,
			Secrets: map[string]string{
				"SAMPLE_SECRET": "mysecretvalue",
			},
		},
		{
			TestCase:      "shared-docker",
			WorkflowPath:  "testdata/exemplar-codecatalyst-action/.codecatalyst/workflows/shared.yaml",
			ExecutionType: runner.ExecutionTypeDocker,
			Secrets: map[string]string{
				"SAMPLE_SECRET": "mysecretvalue",
			},
		},
		{
			TestCase:      "shared-shell",
			WorkflowPath:  "testdata/exemplar-codecatalyst-action/.codecatalyst/workflows/shared.yaml",
			ExecutionType: runner.ExecutionTypeShell,
			Secrets: map[string]string{
				"SAMPLE_SECRET": "mysecretvalue",
			},
		},
	} {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout})
		log.Logger = log.Logger.With().Caller().Stack().Logger()
		ctx = log.Logger.WithContext(ctx)

		tt := tt
		t.Run(tt.TestCase, func(t *testing.T) {
			assert := assert.New(t)
			mockSecrets := newMockSecrets(tt.Secrets)
			err := Run(ctx, &RunParams{
				WorkflowPath: tt.WorkflowPath,
				NewWorkflowFeaturesProviderParams: NewWorkflowFeaturesProviderParams{
					OutputMode:     OutputModeText,
					SecretProvider: mockSecrets,
				},
				NewWorkflowPlansProviderParams: NewWorkflowPlansProviderParams{
					ExecutionType: tt.ExecutionType,
				},
			})
			if err != nil && strings.HasPrefix(err.Error(), "service provider is unavailable:") {
				t.Skip(err.Error())
			}
			if tt.ExpectError != nil {
				assert.Equal(tt.ExpectError, err)
			} else {
				assert.NoError(err)
			}
		})
	}
}

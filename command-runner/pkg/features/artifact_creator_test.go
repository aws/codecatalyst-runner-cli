package features

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/aws/codecatalyst-runner-cli/command-runner/internal/fs"
	"github.com/aws/codecatalyst-runner-cli/command-runner/pkg/runner"

	"github.com/stretchr/testify/assert"
)

func TestArtifactCreator(t *testing.T) {
	assert := assert.New(t)

	artifactDir, err := os.MkdirTemp(fs.TmpDir(), "")
	assert.NoError(err)
	defer os.RemoveAll(artifactDir)

	type TestParams struct {
		TestCase         string
		ArtifactDir      string
		BindToPath       string
		EnvToSet         string
		ArtifactContent  string
		ExpectedFileMaps []*runner.FileMap
		ExpectedEnv      map[string]string
		ExpectedHash     string
	}

	for _, tt := range []*TestParams{
		{
			TestCase:         "Without BindToPath And with EnvToSet",
			ArtifactDir:      filepath.Join(artifactDir, "foo"),
			EnvToSet:         "MYENV",
			ExpectedFileMaps: nil,
			ExpectedEnv: map[string]string{
				"MYENV": filepath.Join(artifactDir, "foo"),
			},
		},
		{
			TestCase:         "Without BindToPath And without EnvToSet",
			ArtifactDir:      filepath.Join(artifactDir, "bar"),
			ExpectedFileMaps: nil,
			ExpectedEnv:      map[string]string{},
		},
		{
			TestCase:    "With BindToPath And EnvToSet",
			ArtifactDir: filepath.Join(artifactDir, "baz"),
			BindToPath:  "/bindpath",
			EnvToSet:    "MYENV",
			ExpectedFileMaps: []*runner.FileMap{
				{
					SourcePath: filepath.Join(artifactDir, "baz"),
					TargetPath: "/bindpath",
					Type:       runner.FileMapTypeBind,
				},
			},
			ExpectedEnv: map[string]string{
				"MYENV": "/bindpath",
			},
		},
		{
			TestCase:         "With artifact content",
			ArtifactDir:      filepath.Join(artifactDir, "test1"),
			ExpectedFileMaps: nil,
			ExpectedEnv:      map[string]string{},
			ArtifactContent:  "Hello world!",
			ExpectedHash:     "e3e98c0ce7a24033f83facbc49f6a765f1d2c371fc31bc94099506ffd927e49f",
		},
	} {
		// setup the code under test
		ctx := context.Background()
		feature := ArtifactCreator(tt.ArtifactDir, tt.BindToPath, tt.EnvToSet)

		// setup the mock
		plan := new(runner.MockPlan)
		m := new(runner.MockPlanExecutor).WithPlan(plan)
		if tt.ArtifactContent != "" {
			m = m.WithExecutor(func(ctx context.Context) error {
				return os.WriteFile(filepath.Join(tt.ArtifactDir, "sample.txt"), []byte(tt.ArtifactContent), 0600)
			})
		}
		m.OnExecute(ctx).Return(nil)

		// run the feature
		err := m.Execute(ctx, feature)

		// assert the results
		assert.NoError(err)
		m.AssertExpectations(t)
		assert.Len(plan.EnvironmentConfiguration().FileMaps, len(tt.ExpectedFileMaps), "%s - FileMap len", tt.TestCase)
		assert.Equal(tt.ExpectedFileMaps, plan.EnvironmentConfiguration().FileMaps, "%s - FileMaps", tt.TestCase)
		assert.Equal(tt.ExpectedEnv, plan.EnvironmentConfiguration().Env, "%s - Env", tt.TestCase)
		if tt.ExpectedHash != "" {
			f, err := os.Open(filepath.Join(artifactDir, "test1.zip"))
			assert.NoError(err)
			defer f.Close()
			hasher := sha256.New()
			_, err = io.Copy(hasher, f)
			assert.NoError(err)
			artifactHash := hex.EncodeToString(hasher.Sum(nil))
			assert.Equal(tt.ExpectedHash, artifactHash, "%s - Artifact hash", tt.TestCase)
		}
	}
}

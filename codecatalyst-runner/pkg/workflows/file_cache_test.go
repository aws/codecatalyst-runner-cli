package workflows

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/aws/codecatalyst-runner-cli/command-runner/pkg/runner"

	"github.com/stretchr/testify/assert"
)

func TestFileCache(t *testing.T) {
	type TestParams struct {
		TestCase         string
		FileCaching      FileCaching
		CreateCacheDir   bool
		ExpectedFileMaps []*runner.FileMap
	}

	mockWorkingDir := "/home/bob/myrepo"
	mockCacheDir, err := os.MkdirTemp("", "mockcachedir")
	assert.NoError(t, err)
	defer os.RemoveAll(mockCacheDir)

	for _, tt := range []*TestParams{
		{
			TestCase: "relative-miss",
			FileCaching: FileCaching{
				"cacheKey1": {
					Path: ".out/file.txt",
				},
			},
			CreateCacheDir: false,
			ExpectedFileMaps: []*runner.FileMap{
				{
					// expect cache to be 'caches/cacheKey1/file1.txt'
					SourcePath: "git/v1/myrepo/.out/file.txt",
					TargetPath: fmt.Sprintf("%s/caches/cacheKey1/", mockCacheDir),
					Type:       runner.FileMapTypeCopyOut,
				},
			},
		},
		{
			TestCase: "relative-hit",
			FileCaching: FileCaching{
				"cacheKey1": {
					Path: ".out/file.txt",
				},
			},
			CreateCacheDir: true,
			ExpectedFileMaps: []*runner.FileMap{
				{
					// expect cache to have 'caches/cacheKey1/file1.txt'
					SourcePath: fmt.Sprintf("%s/caches/cacheKey1/.", mockCacheDir),
					TargetPath: "git/v1/myrepo/.out/",
					Type:       runner.FileMapTypeCopyIn,
				},
				{
					// expect cache to have 'caches/cacheKey1/file1.txt'
					SourcePath: "git/v1/myrepo/.out/file.txt",
					TargetPath: fmt.Sprintf("%s/caches/cacheKey1/", mockCacheDir),
					Type:       runner.FileMapTypeCopyOut,
				},
			},
		},
		{
			TestCase: "relative-dir-miss",
			FileCaching: FileCaching{
				"cacheKey1": {
					Path: ".out/",
				},
			},
			CreateCacheDir: false,
			ExpectedFileMaps: []*runner.FileMap{
				{
					// expect cache to have 'caches/cacheKey1/.out'
					SourcePath: "git/v1/myrepo/.out",
					TargetPath: fmt.Sprintf("%s/caches/cacheKey1/", mockCacheDir),
					Type:       runner.FileMapTypeCopyOut,
				},
			},
		},
		{
			TestCase: "relative-dir-hit",
			FileCaching: FileCaching{
				"cacheKey1": {
					Path: ".out/",
				},
			},
			CreateCacheDir: true,
			ExpectedFileMaps: []*runner.FileMap{
				{
					// expect cache to have 'caches/cacheKey1/.out'
					SourcePath: fmt.Sprintf("%s/caches/cacheKey1/.", mockCacheDir),
					TargetPath: "git/v1/myrepo/",
					Type:       runner.FileMapTypeCopyIn,
				},
				{
					// expect cache to have 'caches/cacheKey1/.out'
					SourcePath: "git/v1/myrepo/.out",
					TargetPath: fmt.Sprintf("%s/caches/cacheKey1/", mockCacheDir),
					Type:       runner.FileMapTypeCopyOut,
				},
			},
		},
		{
			TestCase: "absolute-miss",
			FileCaching: FileCaching{
				"cacheKey1": {
					Path: "/root/path",
				},
			},
			CreateCacheDir: false,
			ExpectedFileMaps: []*runner.FileMap{
				{
					// expect cache to have 'caches/cacheKey1/path'
					SourcePath: "/root/path",
					TargetPath: fmt.Sprintf("%s/caches/cacheKey1/", mockCacheDir),
					Type:       runner.FileMapTypeCopyOut,
				},
			},
		},
		{
			TestCase: "absolute-hit",
			FileCaching: FileCaching{
				"cacheKey1": {
					Path: "/root/path",
				},
			},
			CreateCacheDir: true,
			ExpectedFileMaps: []*runner.FileMap{
				{
					// expect cache to have 'caches/cacheKey1/path'
					SourcePath: fmt.Sprintf("%s/caches/cacheKey1/.", mockCacheDir),
					TargetPath: "/root/",
					Type:       runner.FileMapTypeCopyIn,
				},
				{
					// expect cache to have 'caches/cacheKey1/path'
					SourcePath: "/root/path",
					TargetPath: fmt.Sprintf("%s/caches/cacheKey1/", mockCacheDir),
					Type:       runner.FileMapTypeCopyOut,
				},
			},
		},
	} {
		t.Run(tt.TestCase, func(t *testing.T) {
			assert := assert.New(t)
			// setup the code under test
			ctx := context.Background()

			for key := range tt.FileCaching {
				keyDir := filepath.Join(mockCacheDir, "caches", key)
				os.RemoveAll(keyDir)
				if tt.CreateCacheDir {
					err := os.MkdirAll(keyDir, 0755)
					assert.NoError(err)
				}
			}
			feature := FileCache(mockWorkingDir, tt.FileCaching, staticCacheDirProvider(mockCacheDir))

			// setup the mock
			plan := new(runner.MockPlan)
			m := new(runner.MockPlanExecutor).WithPlan(plan)
			m.OnExecute(ctx).Return(nil)

			// run the feature
			err := m.Execute(ctx, feature)

			assert.NoError(err)
			assert.Equal(tt.ExpectedFileMaps, plan.EnvironmentConfiguration().FileMaps)

			// assert the results
			m.AssertExpectations(t)
		})
	}
}

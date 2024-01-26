package fs

import (
	"os"
	"path/filepath"

	"github.com/rs/zerolog/log"
)

// TmpDir returns the base directory to use for temp files/directories
func TmpDir() string {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		log.Error().Err(err).Msg("Failed to get user cache dir")
		return ""
	}
	tempDir := filepath.Join(cacheDir, "codecatalyst-runner", "tmp")
	if err = os.MkdirAll(tempDir, 0755); err != nil {
		log.Error().Err(err).Msgf("Failed to mkdir: %s", tempDir)
		return ""
	}
	return tempDir
}

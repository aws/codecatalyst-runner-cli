package actions

import (
	"io/fs"
	"os"
	"path/filepath"

	"github.com/rs/zerolog/log"
)

// Find actions recursively under the provided actionSearchPath
// by reading the '.codecatalyst/actions/action.yml file in each directory
func Find(actionSearchPath string) ([]*Action, error) {
	actionSearchPath, err := filepath.Abs(actionSearchPath)
	if err != nil {
		return nil, err
	}
	actions := make([]*Action, 0)
	log.Debug().Msgf("Searching path '%s' for actions", actionSearchPath)
	err = filepath.WalkDir(actionSearchPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.Type().IsDir() {
			if _, err := os.Stat(filepath.Join(path, ".codecatalyst", "actions", "action.yml")); err == nil {
				log.Debug().Msgf("Found action: %s", path)
				action, err := Load(path)
				if err != nil {
					log.Warn().Err(err)
					return nil
				}
				actions = append(actions, action)
				if err != nil {
					return err
				}
				return filepath.SkipDir
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return actions, nil
}

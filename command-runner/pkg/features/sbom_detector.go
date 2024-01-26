package features

import (
	"context"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/aws/codecatalyst-runner-cli/command-runner/pkg/common"
	"github.com/aws/codecatalyst-runner-cli/command-runner/pkg/runner"

	"github.com/rs/zerolog/log"
)

const maxSBOMSize = 10 * 1024 // 10 KB

// SBOMDetector is an [ExecutionFeature] to detect SBOMs created by actions.
// The content of the detected SBOM is loaded into the sbom provided.
func SBOMDetector(directory string, sbom *SBOM) runner.Feature {
	return func(ctx context.Context, plan runner.Plan, e runner.PlanExecutor) error {
		log.Ctx(ctx).Debug().Msg("ENTER SBOMDetector")
		if err := e(ctx); err != nil {
			return err
		}
		err := newSBOMDetector(directory, sbom)(ctx)
		log.Ctx(ctx).Debug().Msg("EXIT SBOMDetector")
		return err
	}
}

func newSBOMDetector(reportDir string, sbom *SBOM) common.Executor {
	return func(ctx context.Context) error {
		return filepath.WalkDir(reportDir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			info, err := d.Info()
			if err != nil {
				return err
			}
			if info.Size() > maxSBOMSize {
				log.Ctx(ctx).Debug().Msgf("Skipping potential SBOM '%s'- too large. %d > %d, path", path, info.Size(), maxSBOMSize)
				return nil
			}

			content, err := os.ReadFile(path)
			if err != nil {
				log.Ctx(ctx).Warn().Msgf("Unable to read potential SBOM '%s': %s", path, err.Error())
				return nil
			}

			data := make(map[string]interface{})
			err = json.Unmarshal(content, &data)
			if err != nil {
				log.Ctx(ctx).Debug().Msgf("Unable to unmarshal potential SBOM '%s': %s", path, err.Error())
				return nil
			}
			for key := range data {
				if strings.EqualFold(key, "spdxVersion") || strings.EqualFold(key, "SPDXID") {
					log.Ctx(ctx).Debug().Msgf("Found SBOM '%s' with type %s", path, sbom.Type)
					sbom.Content = content
					sbom.Type = SBOMTypeSPDX
					break
				}
			}
			return nil
		})
	}
}

// SBOMType - Currently only SPDX is supported
type SBOMType string

const (
	// SBOMTypeSPDX is the SPDX SBOM type
	SBOMTypeSPDX SBOMType = "https://spdx.dev/Document"
)

// SBOM represents a detected SBOM (Software Bill of Materials)
type SBOM struct {
	Type    SBOMType // type of SBOM
	Content []byte   // content of the SBOM
}

package finch

import (
	"context"
	"path/filepath"
	"strings"

	"github.com/aws/codecatalyst-runner-cli/command-runner/internal/containers/types"
	"github.com/aws/codecatalyst-runner-cli/command-runner/pkg/common"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func (fcs *finchContainerService) BuildImage(input types.BuildImageInput) common.Executor {
	return func(ctx context.Context) error {
		logger := log.Ctx(ctx)
		if input.Platform != "" {
			logger.Info().Msgf("🐦 finch build -t %s --platform %s %s", input.ImageTag, input.Platform, input.ContextDir)
		} else {
			logger.Info().Msgf("🐦 finch build -t %s %s", input.ImageTag, input.ContextDir)
		}
		if common.Dryrun(ctx) {
			return nil
		}

		if exists, err := fcs.ImageExistsLocally(ctx, input.ImageTag, input.Platform); exists {
			log.Ctx(ctx).Debug().Msgf("skipping build - image '%s' already exists locally", input.ImageTag)
			return nil
		} else if err != nil {
			return err
		}

		f, err := newFinch(finchInstallDir)
		if err != nil {
			return err
		}

		logger.Debug().Msgf("Building image from '%v'", input.ContextDir)

		args := []string{"build", "--tag", input.ImageTag, "--file", filepath.Join(input.ContextDir, input.Dockerfile), "--rm"}
		if input.Platform != "" {
			args = append(args, "--platform", input.Platform)
		}
		args = append(args, input.ContextDir)

		logger.Debug().Msgf("Creating image from context dir '%s' with tag '%s' and platform '%s'", input.ContextDir, input.ImageTag, input.Platform)
		ldebug := &logWriter{
			logger: logger,
			level:  zerolog.DebugLevel,
		}
		return f.RunWithStdio(ctx, nil, ldebug, ldebug, args...)
	}
}

type logWriter struct {
	logger *zerolog.Logger
	level  zerolog.Level
}

func (l *logWriter) Write(p []byte) (int, error) {
	if len(p) > 0 {
		l.logger.WithLevel(l.level).Msg(strings.TrimSpace(string(p)))
	}
	return len(p), nil
}

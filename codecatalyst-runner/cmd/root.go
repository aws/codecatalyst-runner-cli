package cmd

import (
	"strings"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

type commonParams struct {
	Version string
	Verbose bool
}

// NewRootCmd adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func NewRootCmd(version string) *cobra.Command {
	version = strings.TrimSpace(version)
	params := new(commonParams)
	params.Version = version

	var rootCmd = &cobra.Command{
		Use:              "ccr",
		Short:            "Tool to run codecatalyst locally",
		PersistentPreRun: setup(params),
		Version:          version,
		SilenceUsage:     true,
		SilenceErrors:    true,
	}
	rootCmd.InitDefaultVersionFlag()
	rootCmd.PersistentFlags().BoolVarP(&params.Verbose, "verbose", "V", false, "verbose output")
	setupExecuteCommands(rootCmd)
	return rootCmd
}

func setup(params *commonParams) func(*cobra.Command, []string) {
	return func(cmd *cobra.Command, _ []string) {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
		if params.Verbose {
			zerolog.SetGlobalLevel(zerolog.DebugLevel)
		}
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: cmd.OutOrStdout()})
		if params.Verbose {
			log.Logger = log.Logger.With().Caller().Stack().Logger()
		}

		ctx := log.Logger.WithContext(cmd.Context())
		cmd.SetContext(ctx)
	}
}

package cmd

import (
	"os"
	"runtime"

	"github.com/aws/codecatalyst-runner-cli/codecatalyst-runner/pkg/workflows"

	"github.com/aws/codecatalyst-runner-cli/command-runner/pkg/runner"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

func setupExecuteCommands(rootCmd *cobra.Command) {
	params := new(workflows.RunParams)

	var defaultOutputMode workflows.OutputMode
	if os.Getenv("CI") != "true" && term.IsTerminal(int(os.Stdout.Fd())) {
		defaultOutputMode = workflows.OutputModeTUI
	} else {
		defaultOutputMode = workflows.OutputModeText
	}
	rootCmd.PersistentFlags().BoolVarP(&params.Reuse, "reuse", "R", false, "Reuse containers between executions")
	rootCmd.PersistentFlags().StringVarP(&params.WorkingDir, "working-dir", "w", ".", "directory to run workflow against")
	rootCmd.PersistentFlags().StringVarP(&params.WorkflowPath, "workflow-file", "f", "", "path to workflow to run")
	rootCmd.PersistentFlags().StringVarP(&params.Action, "action", "a", "", "action to run (default: *)")
	rootCmd.PersistentFlags().BoolVarP(&params.BindWorkingDir, "bind", "b", false, "bind working directory rather than create a copy")
	rootCmd.PersistentFlags().BoolVarP(&params.NoOutput, "quiet", "q", false, "disable logging of output from actions")
	rootCmd.PersistentFlags().BoolVarP(&params.Dryrun, "dryrun", "n", false, "dry run")
	rootCmd.PersistentFlags().BoolVarP(&params.NoCache, "no-cache", "C", false, "disable file caches")
	rootCmd.PersistentFlags().StringVarP((*string)(&params.ExecutionType), "executor", "x", string(runner.DefaultExecutionType()), "executor type [docker,finch,shell]")
	rootCmd.PersistentFlags().IntVarP(&params.Concurrency, "concurrency", "c", runtime.NumCPU(), "number of policies to execute concurrently")
	rootCmd.PersistentFlags().StringToStringVarP(&params.EnvironmentProfiles, "environments", "e", make(map[string]string), "map workflow environment names to AWS CLI profile names")
	rootCmd.PersistentFlags().StringVarP((*string)(&params.OutputMode), "output-format", "t", string(defaultOutputMode), "output mode [tui,text]")

	executeCommand := func(cmd *cobra.Command, args []string) error {
		if len(args) == 1 {
			params.WorkflowName = args[0]
		}
		ctx := cmd.Context()
		err := workflows.Run(ctx, params)
		log.Ctx(ctx).Debug().Err(err).Msg("execute complete")
		return err
	}
	rootCmd.AddCommand(&cobra.Command{
		Use:     "execute",
		Aliases: []string{"exec"},
		Short:   "Execute actions",
		RunE:    executeCommand,
		Args:    cobra.MaximumNArgs(1),
	})
	rootCmd.RunE = executeCommand
	rootCmd.Args = cobra.MaximumNArgs(1)
}

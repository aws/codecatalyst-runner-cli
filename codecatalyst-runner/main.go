package main

import (
	"context"
	_ "embed"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/aws/codecatalyst-runner-cli/codecatalyst-runner/cmd"

	"github.com/rs/zerolog/log"
)

//go:embed VERSION
var version string

func withSignalHandler(ctx context.Context, signals ...os.Signal) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(ctx)
	c := make(chan os.Signal, 1)
	signal.Notify(c, signals...)
	go func() {
		select {
		case <-c:
			log.Ctx(ctx).Debug().Msg("received signal, shutting down")
			cancel()
		case <-ctx.Done():
		}
	}()
	return ctx, func() {
		signal.Stop(c)
		cancel()
	}
}

func main() {
	ctx := context.Background()
	ctx, close := withSignalHandler(ctx, os.Interrupt, syscall.SIGTERM, syscall.SIGPIPE)

	rootCmd := cmd.NewRootCmd(version)
	err := rootCmd.ExecuteContext(ctx)
	close()
	if err != nil {
		if err != context.Canceled {
			fmt.Printf("error executing command: %s\n", err.Error())
		}
		os.Exit(1)
	}
}

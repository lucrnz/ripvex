package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/lucrnz/ripvex/internal/cleanup"
	"github.com/lucrnz/ripvex/internal/cli"
)

func main() {
	// Set up signal handling for graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Create cleanup tracker for temporary files
	tracker := cleanup.NewTracker()
	defer tracker.Cleanup()

	// Run CLI with context
	if err := cli.ExecuteContext(ctx, tracker); err != nil {
		// Check if error is due to context cancellation (interrupt)
		if ctx.Err() == context.Canceled {
			fmt.Fprintln(os.Stderr, "\nInterrupted")
			os.Exit(130) // Standard exit code for SIGINT
		}
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

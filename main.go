// Package main is the entry point for sortie, an AI-powered CLI tool
// for generating and managing code changes through intelligent agents.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/sortie-ai/sortie/internal/cli"
	"github.com/sortie-ai/sortie/internal/version"
)

func main() {
	// Set up context that cancels on interrupt signals
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle graceful shutdown on SIGINT and SIGTERM
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		select {
		case sig := <-sigCh:
			fmt.Fprintf(os.Stderr, "\nReceived signal %s, shutting down...\n", sig)
			cancel()
		case <-ctx.Done():
		}
	}()

	// Print version info when running in debug mode
	if os.Getenv("SORTIE_DEBUG") == "1" {
		fmt.Fprintf(os.Stderr, "sortie %s\n", version.Version)
	}

	// Execute the root CLI command
	if err := cli.Execute(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

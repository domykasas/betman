// Package main provides the CLI interface for the coin flip game.
package main

import (
	"context"
	"fmt"
	"os"

	"coinflip-game/cmd/cli/commands"
	"coinflip-game/internal/config"
	"coinflip-game/internal/logger"

	"go.uber.org/zap"
)

func main() {
	// Load configuration
	cfg, err := config.Load("")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Initialize logger
	log, err := logger.New(cfg.Logging.Level, cfg.Logging.Development)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer log.Sync()

	// Create and execute root command
	ctx := context.Background()
	rootCmd := commands.NewRootCommand(cfg, log)

	if err := rootCmd.ExecuteContext(ctx); err != nil {
		log.Error("Command execution failed", zap.Error(err))
		os.Exit(1)
	}
}

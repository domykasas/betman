//go:build !gui && !server

// main.go is the CLI entry point for the multiplayer coin flip game.
// This provides both single-player and multiplayer CLI functionality.
package main

import (
	"fmt"
	"os"

	"coinflip-game/internal/config"
	"coinflip-game/internal/logger"
	"coinflip-game/cmd/cli/commands"
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
	rootCmd := commands.NewRootCommand(cfg, log)
	
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
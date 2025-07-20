// Package commands provides the CLI command structure for the coin flip game.
package commands

import (
	"github.com/spf13/cobra"
	"go.uber.org/zap"

	"coinflip-game/internal/config"
	"coinflip-game/internal/game"
	"coinflip-game/internal/storage"
)

// CLIApp holds the application dependencies for CLI commands
type CLIApp struct {
	Config *config.Config
	Engine *game.Engine
	Logger *zap.Logger
	Repo   *storage.MemoryRepository
}

// NewRootCommand creates the root CLI command with all subcommands
func NewRootCommand(cfg *config.Config, logger *zap.Logger) *cobra.Command {
	// Initialize dependencies
	repo := storage.NewMemoryRepository()
	rng := game.NewDefaultRandomGenerator()
	engine := game.NewEngine(cfg.ToGameConfig(), repo, rng, logger)

	app := &CLIApp{
		Config: cfg,
		Engine: engine,
		Logger: logger,
		Repo:   repo,
	}

	rootCmd := &cobra.Command{
		Use:   "coinflip",
		Short: "A coin flip betting game",
		Long: `Coinflip is a simple yet educational gambling game where you bet on the outcome 
of a coin flip. You can place bets on either heads or tails, and if you guess correctly, 
you win based on the configured payout ratio.

This implementation demonstrates clean Go architecture, dependency injection, 
comprehensive testing, and modern development practices.`,
		Example: `  # Start an interactive game session
  coinflip play

  # Place a specific bet
  coinflip bet --amount 10 --choice heads

  # Check your balance and statistics
  coinflip status

  # View game history
  coinflip history`,
	}

	// Add subcommands
	rootCmd.AddCommand(
		newPlayCommand(app),
		newBetCommand(app),
		newStatusCommand(app),
		newHistoryCommand(app),
		newConfigCommand(app),
	)

	return rootCmd
}

// getPlayerID returns a default player ID for single-player CLI mode
func getPlayerID() string {
	return "cli_player"
}

package commands

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

// newStatusCommand creates the status command for displaying player information
func newStatusCommand(app *CLIApp) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Display current player status and statistics",
		Long: `Display comprehensive information about the current player including 
balance, game statistics, and current bet status.`,
		Example: `  coinflip status`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return showPlayerStatus(cmd.Context(), app)
		},
	}
}

// showPlayerStatus displays comprehensive player information
func showPlayerStatus(ctx context.Context, app *CLIApp) error {
	playerID := getPlayerID()

	// Get player info
	player, err := app.Engine.GetPlayer(ctx, playerID)
	if err != nil {
		return fmt.Errorf("failed to get player: %w", err)
	}

	fmt.Println("👤 Player Status")
	fmt.Println("================")
	fmt.Printf("Player ID: %s\n", player.ID)
	fmt.Printf("💰 Balance: $%.2f\n", player.Balance)

	// Show game configuration
	config := app.Engine.GetConfig()
	fmt.Printf("🎯 Min bet: $%.2f\n", config.MinBet)
	fmt.Printf("🎯 Max bet: $%.2f\n", config.MaxBet)
	fmt.Printf("💎 Payout ratio: %.1fx\n", config.PayoutRatio)

	// Check if player can play
	if player.Balance < config.MinBet {
		fmt.Printf("🚫 Cannot play: balance below minimum bet\n")
	} else {
		fmt.Printf("✅ Can play: balance sufficient for betting\n")
	}

	// Show current bet if any
	if currentBet := app.Engine.GetCurrentBet(); currentBet != nil {
		fmt.Printf("\n🎲 Active Bet\n")
		fmt.Printf("Amount: $%.2f\n", currentBet.Amount)
		fmt.Printf("Choice: %s\n", currentBet.Choice)
		fmt.Printf("Placed: %s\n", currentBet.Timestamp.Format("2006-01-02 15:04:05"))
	}

	// Show statistics
	fmt.Printf("\n📊 Statistics\n")
	fmt.Println("=============")
	displayStats(&player.Stats)

	return nil
}

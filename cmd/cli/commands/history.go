package commands

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"coinflip-game/internal/game"
)

// newHistoryCommand creates the history command for viewing game results
func newHistoryCommand(app *CLIApp) *cobra.Command {
	var limit int

	cmd := &cobra.Command{
		Use:   "history",
		Short: "Display recent game history",
		Long: `Display a list of recent game results including the coin flip outcome, 
bet details, and winnings. Results are shown in reverse chronological order 
(most recent first).`,
		Example: `  coinflip history
  coinflip history --limit 5`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return showGameHistory(cmd.Context(), app, limit)
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "l", 10, "Maximum number of results to show")

	return cmd
}

// showGameHistory displays recent game results
func showGameHistory(ctx context.Context, app *CLIApp, limit int) error {
	results, err := app.Engine.GetGameHistory(ctx, limit)
	if err != nil {
		return fmt.Errorf("failed to get game history: %w", err)
	}

	if len(results) == 0 {
		fmt.Println("📭 No game history found. Play some games first!")
		return nil
	}

	fmt.Printf("📜 Game History (last %d games)\n", len(results))
	fmt.Println("================================")

	for i, result := range results {
		displayHistoryEntry(i+1, result)
		if i < len(results)-1 {
			fmt.Println(strings.Repeat("-", 40))
		}
	}

	return nil
}

// displayHistoryEntry shows a single game result in the history
func displayHistoryEntry(index int, result *game.Result) {
	coinEmoji := "🟡"
	if result.Side == game.Heads {
		coinEmoji = "👑"
	} else {
		coinEmoji = "🦅"
	}

	// Header with game number and result
	fmt.Printf("🎯 Game #%d: %s %s\n", index, coinEmoji, strings.ToUpper(string(result.Side)))
	fmt.Printf("⏰ Time: %s\n", result.Timestamp.Format("2006-01-02 15:04:05"))

	// Bet details if available
	if result.Bet != nil {
		fmt.Printf("💸 Bet: $%.2f on %s\n", result.Bet.Amount, strings.ToUpper(string(result.Bet.Choice)))
	}

	// Outcome
	if result.Won {
		fmt.Printf("✅ Won: $%.2f", result.Payout)
		if result.Bet != nil {
			profit := result.Payout - result.Bet.Amount
			fmt.Printf(" (profit: +$%.2f)", profit)
		}
		fmt.Println()
	} else {
		fmt.Printf("❌ Lost")
		if result.Bet != nil {
			fmt.Printf(": -$%.2f", result.Bet.Amount)
		}
		fmt.Println()
	}

	// Seed for verification
	if result.Seed != "" {
		fmt.Printf("🔍 Seed: %s\n", result.Seed[:16]+"...") // Show first 16 chars
	}
}

package commands

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"coinflip-game/internal/game"
)

// newBetCommand creates the bet command for placing a single bet
func newBetCommand(app *CLIApp) *cobra.Command {
	var amount float64
	var choice string

	cmd := &cobra.Command{
		Use:   "bet",
		Short: "Place a single bet and flip the coin",
		Long: `Place a single bet on heads or tails and immediately flip the coin 
to see the result. This is useful for scripting or one-off bets.`,
		Example: `  coinflip bet --amount 10 --choice heads
  coinflip bet -a 25.5 -c tails`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSingleBet(cmd.Context(), app, amount, choice)
		},
	}

	cmd.Flags().Float64VarP(&amount, "amount", "a", 0, "Bet amount (required)")
	cmd.Flags().StringVarP(&choice, "choice", "c", "", "Choice: heads or tails (required)")

	cmd.MarkFlagRequired("amount")
	cmd.MarkFlagRequired("choice")

	return cmd
}

// runSingleBet executes a single bet operation
func runSingleBet(ctx context.Context, app *CLIApp, amount float64, choiceStr string) error {
	playerID := getPlayerID()

	// Validate and parse choice
	var choice game.Side
	switch choiceStr {
	case "heads", "h":
		choice = game.Heads
	case "tails", "t":
		choice = game.Tails
	default:
		return fmt.Errorf("invalid choice '%s', must be 'heads' or 'tails'", choiceStr)
	}

	// Get player info
	player, err := app.Engine.GetPlayer(ctx, playerID)
	if err != nil {
		return fmt.Errorf("failed to get player: %w", err)
	}

	fmt.Printf("💰 Current balance: $%.2f\n", player.Balance)

	// Check for existing bet
	if currentBet := app.Engine.GetCurrentBet(); currentBet != nil {
		return fmt.Errorf("you already have an active bet of $%.2f on %s, please flip the coin first",
			currentBet.Amount, currentBet.Choice)
	}

	// Place bet
	bet, err := app.Engine.PlaceBet(ctx, playerID, amount, choice)
	if err != nil {
		return fmt.Errorf("failed to place bet: %w", err)
	}

	fmt.Printf("✅ Bet placed: $%.2f on %s\n", bet.Amount, bet.Choice)
	fmt.Println("🎲 Flipping coin...")

	// Flip the coin
	result, err := app.Engine.FlipCoin(ctx, playerID)
	if err != nil {
		return fmt.Errorf("failed to flip coin: %w", err)
	}

	// Display result
	displayResult(result)

	// Get updated balance
	player, err = app.Engine.GetPlayer(ctx, playerID)
	if err != nil {
		return fmt.Errorf("failed to get updated player info: %w", err)
	}

	fmt.Printf("\n💰 New balance: $%.2f\n", player.Balance)
	return nil
}

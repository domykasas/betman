package commands

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"go.uber.org/zap"

	"coinflip-game/internal/game"
)

// newPlayCommand creates the interactive play command
func newPlayCommand(app *CLIApp) *cobra.Command {
	return &cobra.Command{
		Use:   "play",
		Short: "Start an interactive coin flip game session",
		Long: `Start an interactive session where you can place multiple bets, 
view your balance, and play continuously until you choose to quit.`,
		Example: `  coinflip play`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInteractiveGame(cmd.Context(), app)
		},
	}
}

// runInteractiveGame runs the main interactive game loop
func runInteractiveGame(ctx context.Context, app *CLIApp) error {
	playerID := getPlayerID()
	scanner := bufio.NewScanner(os.Stdin)

	// Get or create player
	player, err := app.Engine.GetPlayer(ctx, playerID)
	if err != nil {
		return fmt.Errorf("failed to get player: %w", err)
	}

	fmt.Println("🪙 Welcome to Coin Flip!")
	fmt.Println("========================")
	fmt.Printf("Starting balance: $%.2f\n", player.Balance)
	fmt.Printf("Minimum bet: $%.2f, Maximum bet: $%.2f\n", app.Config.Game.MinBet, app.Config.Game.MaxBet)
	fmt.Printf("Payout ratio: %.1fx\n", app.Config.Game.PayoutRatio)
	fmt.Println()

	for {
		// Check if player can continue playing
		player, err = app.Engine.GetPlayer(ctx, playerID)
		if err != nil {
			return fmt.Errorf("failed to get player: %w", err)
		}

		if player.Balance < app.Config.Game.MinBet {
			fmt.Printf("🚫 Game Over! Your balance ($%.2f) is below the minimum bet ($%.2f)\n",
				player.Balance, app.Config.Game.MinBet)
			break
		}

		// Show current status
		fmt.Printf("💰 Current balance: $%.2f\n", player.Balance)

		// Check for active bet
		currentBet := app.Engine.GetCurrentBet()
		if currentBet != nil {
			fmt.Printf("🎲 Active bet: $%.2f on %s\n", currentBet.Amount, currentBet.Choice)
			fmt.Print("Press Enter to flip the coin, or type 'cancel' to cancel the bet: ")

			if !scanner.Scan() {
				break
			}

			input := strings.TrimSpace(scanner.Text())
			if strings.ToLower(input) == "cancel" {
				if err := app.Engine.CancelCurrentBet(ctx, playerID); err != nil {
					fmt.Printf("❌ Failed to cancel bet: %v\n", err)
					continue
				}
				fmt.Println("✅ Bet cancelled and refunded.")
				continue
			}

			// Flip the coin
			result, err := app.Engine.FlipCoin(ctx, playerID)
			if err != nil {
				fmt.Printf("❌ Failed to flip coin: %v\n", err)
				continue
			}

			displayResult(result)
			continue
		}

		// Prompt for new bet
		fmt.Print("💸 Enter bet amount (or 'quit' to exit): $")
		if !scanner.Scan() {
			break
		}

		input := strings.TrimSpace(scanner.Text())
		if strings.ToLower(input) == "quit" || strings.ToLower(input) == "q" {
			break
		}

		// Parse bet amount
		amount, err := strconv.ParseFloat(input, 64)
		if err != nil {
			fmt.Printf("❌ Invalid amount: %v\n", err)
			continue
		}

		// Get choice
		fmt.Print("🪙 Choose heads (h) or tails (t): ")
		if !scanner.Scan() {
			break
		}

		choiceInput := strings.ToLower(strings.TrimSpace(scanner.Text()))
		var choice game.Side
		switch choiceInput {
		case "h", "heads":
			choice = game.Heads
		case "t", "tails":
			choice = game.Tails
		default:
			fmt.Println("❌ Invalid choice. Please enter 'h' for heads or 't' for tails.")
			continue
		}

		// Place bet
		bet, err := app.Engine.PlaceBet(ctx, playerID, amount, choice)
		if err != nil {
			fmt.Printf("❌ Failed to place bet: %v\n", err)
			continue
		}

		fmt.Printf("✅ Bet placed: $%.2f on %s\n", bet.Amount, bet.Choice)
		fmt.Print("🎲 Press Enter to flip the coin...")
		scanner.Scan()

		// Flip the coin
		result, err := app.Engine.FlipCoin(ctx, playerID)
		if err != nil {
			fmt.Printf("❌ Failed to flip coin: %v\n", err)
			continue
		}

		displayResult(result)
		fmt.Println()
	}

	// Show final stats
	fmt.Println("\n📊 Final Statistics:")
	stats, err := app.Repo.GetStats(ctx, playerID)
	if err != nil {
		app.Logger.Error("Failed to get final stats", zap.Error(err))
	} else {
		displayStats(stats)
	}

	fmt.Println("👋 Thanks for playing!")
	return nil
}

// displayResult shows the result of a coin flip in a formatted way
func displayResult(result *game.Result) {
	coinEmoji := "🟡"
	if result.Side == game.Heads {
		coinEmoji = "👑"
	} else {
		coinEmoji = "🦅"
	}

	fmt.Printf("\n🎯 Coin flip result: %s %s\n", coinEmoji, strings.ToUpper(string(result.Side)))

	if result.Won {
		fmt.Printf("🎉 You won! Payout: $%.2f\n", result.Payout)
		if result.Bet != nil {
			profit := result.Payout - result.Bet.Amount
			fmt.Printf("💵 Profit: +$%.2f\n", profit)
		}
	} else {
		fmt.Printf("😞 You lost! Better luck next time.\n")
		if result.Bet != nil {
			fmt.Printf("💸 Loss: -$%.2f\n", result.Bet.Amount)
		}
	}
}

// displayStats shows player statistics in a formatted way
func displayStats(stats *game.Stats) {
	fmt.Printf("Games played: %d\n", stats.GamesPlayed)
	fmt.Printf("Games won: %d\n", stats.GamesWon)
	fmt.Printf("Win rate: %.1f%%\n", stats.WinRate)
	fmt.Printf("Total wagered: $%.2f\n", stats.TotalWagered)
	fmt.Printf("Total winnings: $%.2f\n", stats.TotalWinnings)
	fmt.Printf("Net profit: $%.2f\n", stats.NetProfit)
}

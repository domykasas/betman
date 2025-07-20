package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

// newConfigCommand creates the config command for displaying configuration
func newConfigCommand(app *CLIApp) *cobra.Command {
	return &cobra.Command{
		Use:   "config",
		Short: "Display current game configuration",
		Long: `Display the current game configuration including betting limits, 
payout ratios, and other game settings.`,
		Example: `  coinflip config`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return showConfiguration(app)
		},
	}
}

// showConfiguration displays the current game configuration
func showConfiguration(app *CLIApp) error {
	fmt.Println("⚙️  Game Configuration")
	fmt.Println("======================")

	// Game settings
	fmt.Println("🎯 Game Settings:")
	fmt.Printf("  Starting balance: $%.2f\n", app.Config.Game.StartingBalance)
	fmt.Printf("  Minimum bet: $%.2f\n", app.Config.Game.MinBet)
	fmt.Printf("  Maximum bet: $%.2f\n", app.Config.Game.MaxBet)
	fmt.Printf("  Payout ratio: %.1fx\n", app.Config.Game.PayoutRatio)

	// Logging settings
	fmt.Println("\n📝 Logging Settings:")
	fmt.Printf("  Level: %s\n", app.Config.Logging.Level)
	fmt.Printf("  Development mode: %t\n", app.Config.Logging.Development)

	// UI settings
	fmt.Println("\n🖥️  UI Settings:")
	fmt.Printf("  Theme: %s\n", app.Config.UI.Theme)
	fmt.Printf("  Window size: %dx%d\n", app.Config.UI.WindowWidth, app.Config.UI.WindowHeight)

	// Configuration tips
	fmt.Println("\n💡 Configuration Tips:")
	fmt.Println("  • Edit configs/config.json to change settings")
	fmt.Println("  • Use environment variables with COINFLIP_ prefix")
	fmt.Println("  • Example: COINFLIP_GAME_MIN_BET=5.0")

	return nil
}

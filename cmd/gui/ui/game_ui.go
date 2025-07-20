// Package ui provides the graphical user interface components for the coin flip game.
package ui

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"go.uber.org/zap"

	"coinflip-game/internal/config"
	"coinflip-game/internal/game"
)

// GameUI manages the main game interface
type GameUI struct {
	ctx      context.Context
	app      fyne.App
	window   fyne.Window
	engine   *game.Engine
	config   *config.Config
	logger   *zap.Logger
	playerID string

	// UI components
	balanceLabel   *widget.Label
	betAmountEntry *widget.Entry
	headsButton    *widget.Button
	tailsButton    *widget.Button
	flipButton     *widget.Button
	cancelButton   *widget.Button
	resultLabel    *widget.Label
	statusLabel    *widget.Label
	historyList    *widget.List
	statsContainer *fyne.Container

	// Game state
	currentBet  *game.Bet
	gameHistory []*game.Result
}

// NewGameUI creates a new game UI instance
func NewGameUI(ctx context.Context, app fyne.App, engine *game.Engine, cfg *config.Config, logger *zap.Logger) *GameUI {
	ui := &GameUI{
		ctx:      ctx,
		app:      app,
		engine:   engine,
		config:   cfg,
		logger:   logger,
		playerID: "gui_player",
	}

	ui.window = app.NewWindow("🪙 Coin Flip Game")
	ui.setupUI()
	ui.refreshPlayerInfo()

	return ui
}

// GetWindow returns the main application window
func (ui *GameUI) GetWindow() fyne.Window {
	return ui.window
}

// setupUI creates and arranges all UI components
func (ui *GameUI) setupUI() {
	// Player info section
	ui.balanceLabel = widget.NewLabel("Balance: $0.00")
	ui.balanceLabel.TextStyle = fyne.TextStyle{Bold: true}

	// Betting section
	ui.betAmountEntry = widget.NewEntry()
	ui.betAmountEntry.SetPlaceHolder("Enter bet amount...")
	ui.betAmountEntry.Validator = func(s string) error {
		if s == "" {
			return nil // Allow empty for placeholder
		}
		amount, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return fmt.Errorf("invalid number")
		}
		if amount < ui.config.Game.MinBet || amount > ui.config.Game.MaxBet {
			return fmt.Errorf("bet must be between $%.2f and $%.2f",
				ui.config.Game.MinBet, ui.config.Game.MaxBet)
		}
		return nil
	}

	ui.headsButton = widget.NewButton("👑 Heads", func() {
		ui.placeBet(game.Heads)
	})
	ui.tailsButton = widget.NewButton("🦅 Tails", func() {
		ui.placeBet(game.Tails)
	})

	ui.flipButton = widget.NewButton("🎲 Flip Coin!", func() {
		ui.flipCoin()
	})
	ui.flipButton.Importance = widget.HighImportance

	ui.cancelButton = widget.NewButton("❌ Cancel Bet", func() {
		ui.cancelBet()
	})

	bettingForm := container.NewVBox(
		widget.NewLabel("💸 Place Your Bet"),
		ui.betAmountEntry,
		container.NewGridWithColumns(2, ui.headsButton, ui.tailsButton),
	)

	actionContainer := container.NewVBox(
		ui.flipButton,
		ui.cancelButton,
	)

	// Result section
	ui.resultLabel = widget.NewLabel("🎯 Place a bet to start playing!")
	ui.resultLabel.TextStyle = fyne.TextStyle{Bold: true}
	ui.resultLabel.Alignment = fyne.TextAlignCenter

	ui.statusLabel = widget.NewLabel("Ready to play")

	// Statistics section
	ui.statsContainer = container.NewVBox(
		widget.NewLabel("📊 Statistics"),
	)

	// History section
	ui.gameHistory = make([]*game.Result, 0)
	ui.historyList = widget.NewList(
		func() int {
			return len(ui.gameHistory)
		},
		func() fyne.CanvasObject {
			return container.NewHBox(
				widget.NewLabel("Game"),
				widget.NewLabel("Result"),
				widget.NewLabel("Outcome"),
			)
		},
		func(id widget.ListItemID, item fyne.CanvasObject) {
			if id >= len(ui.gameHistory) {
				return
			}
			result := ui.gameHistory[id]
			cont := item.(*fyne.Container)

			// Game info
			gameLabel := cont.Objects[0].(*widget.Label)
			gameLabel.SetText(fmt.Sprintf("#%d", len(ui.gameHistory)-id))

			// Result
			resultLabel := cont.Objects[1].(*widget.Label)
			coinEmoji := "👑"
			if result.Side == game.Tails {
				coinEmoji = "🦅"
			}
			resultLabel.SetText(fmt.Sprintf("%s %s", coinEmoji, strings.ToUpper(string(result.Side))))

			// Outcome
			outcomeLabel := cont.Objects[2].(*widget.Label)
			if result.Won {
				outcomeLabel.SetText(fmt.Sprintf("✅ +$%.2f", result.Payout-result.Bet.Amount))
			} else {
				outcomeLabel.SetText(fmt.Sprintf("❌ -$%.2f", result.Bet.Amount))
			}
		},
	)

	// Layout
	leftPanel := container.NewVBox(
		ui.balanceLabel,
		widget.NewSeparator(),
		bettingForm,
		widget.NewSeparator(),
		actionContainer,
		widget.NewSeparator(),
		ui.resultLabel,
		ui.statusLabel,
	)

	rightPanel := container.NewVBox(
		ui.statsContainer,
		widget.NewSeparator(),
		widget.NewLabel("📜 Recent Games"),
		container.NewScroll(ui.historyList),
	)

	content := container.NewHSplit(leftPanel, rightPanel)
	content.SetOffset(0.6) // 60% left, 40% right

	ui.window.SetContent(content)
	ui.updateButtonStates()
}

// refreshPlayerInfo updates the player information display
func (ui *GameUI) refreshPlayerInfo() {
	player, err := ui.engine.GetPlayer(ui.ctx, ui.playerID)
	if err != nil {
		ui.logger.Error("Failed to get player info", zap.Error(err))
		ui.statusLabel.SetText("Error loading player info")
		return
	}

	ui.balanceLabel.SetText(fmt.Sprintf("💰 Balance: $%.2f", player.Balance))
	ui.updateStats(&player.Stats)
	ui.updateButtonStates()
}

// updateStats refreshes the statistics display
func (ui *GameUI) updateStats(stats *game.Stats) {
	ui.statsContainer.RemoveAll()

	ui.statsContainer.Add(widget.NewLabel("📊 Statistics"))
	ui.statsContainer.Add(widget.NewLabel(fmt.Sprintf("Games: %d", stats.GamesPlayed)))
	ui.statsContainer.Add(widget.NewLabel(fmt.Sprintf("Won: %d", stats.GamesWon)))
	ui.statsContainer.Add(widget.NewLabel(fmt.Sprintf("Win Rate: %.1f%%", stats.WinRate)))
	ui.statsContainer.Add(widget.NewLabel(fmt.Sprintf("Wagered: $%.2f", stats.TotalWagered)))
	ui.statsContainer.Add(widget.NewLabel(fmt.Sprintf("Winnings: $%.2f", stats.TotalWinnings)))
	ui.statsContainer.Add(widget.NewLabel(fmt.Sprintf("Net: $%.2f", stats.NetProfit)))
}

// updateButtonStates enables/disables buttons based on game state
func (ui *GameUI) updateButtonStates() {
	ui.currentBet = ui.engine.GetCurrentBet()

	hasBet := ui.currentBet != nil
	validAmount := ui.betAmountEntry.Validate() == nil && ui.betAmountEntry.Text != ""

	// Disable betting buttons if we have an active bet
	ui.headsButton.Enable()
	ui.tailsButton.Enable()
	ui.betAmountEntry.Enable()

	if hasBet {
		ui.headsButton.Disable()
		ui.tailsButton.Disable()
		ui.betAmountEntry.Disable()
	}

	// Enable/disable action buttons
	if hasBet {
		ui.flipButton.Enable()
		ui.cancelButton.Enable()
		ui.statusLabel.SetText(fmt.Sprintf("🎲 Bet placed: $%.2f on %s",
			ui.currentBet.Amount, ui.currentBet.Choice))
	} else {
		ui.flipButton.Disable()
		ui.cancelButton.Disable()
		if validAmount {
			ui.statusLabel.SetText("🎯 Choose heads or tails")
		} else {
			ui.statusLabel.SetText("💸 Enter a valid bet amount")
		}
	}
}

// placeBet handles placing a new bet
func (ui *GameUI) placeBet(choice game.Side) {
	if ui.currentBet != nil {
		dialog.ShowInformation("Active Bet", "You already have an active bet. Flip the coin or cancel it first.", ui.window)
		return
	}

	amountStr := ui.betAmountEntry.Text
	amount, err := strconv.ParseFloat(amountStr, 64)
	if err != nil {
		dialog.ShowError(fmt.Errorf("invalid bet amount: %v", err), ui.window)
		return
	}

	bet, err := ui.engine.PlaceBet(ui.ctx, ui.playerID, amount, choice)
	if err != nil {
		dialog.ShowError(fmt.Errorf("failed to place bet: %v", err), ui.window)
		return
	}

	ui.logger.Info("Bet placed via GUI",
		zap.String("bet_id", bet.ID),
		zap.Float64("amount", amount),
		zap.String("choice", choice.String()),
	)

	ui.refreshPlayerInfo()
	ui.resultLabel.SetText("🎲 Bet placed! Click 'Flip Coin' to play.")
}

// flipCoin executes the coin flip
func (ui *GameUI) flipCoin() {
	if ui.currentBet == nil {
		dialog.ShowInformation("No Bet", "Please place a bet first.", ui.window)
		return
	}

	// Show flipping animation
	ui.resultLabel.SetText("🌀 Flipping coin...")
	ui.flipButton.Disable()
	ui.cancelButton.Disable()

	// Simulate coin flip delay for better UX
	go func() {
		time.Sleep(1 * time.Second)

		result, err := ui.engine.FlipCoin(ui.ctx, ui.playerID)
		if err != nil {
			fyne.CurrentApp().SendNotification(&fyne.Notification{
				Title:   "Error",
				Content: fmt.Sprintf("Failed to flip coin: %v", err),
			})
			ui.updateButtonStates()
			return
		}

		// Update UI on main thread
		ui.showResult(result)
		ui.addToHistory(result)
		ui.refreshPlayerInfo()
	}()
}

// cancelBet cancels the current bet
func (ui *GameUI) cancelBet() {
	if ui.currentBet == nil {
		return
	}

	err := ui.engine.CancelCurrentBet(ui.ctx, ui.playerID)
	if err != nil {
		dialog.ShowError(fmt.Errorf("failed to cancel bet: %v", err), ui.window)
		return
	}

	ui.refreshPlayerInfo()
	ui.resultLabel.SetText("✅ Bet cancelled and refunded")
}

// showResult displays the game result
func (ui *GameUI) showResult(result *game.Result) {
	coinEmoji := "👑"
	if result.Side == game.Tails {
		coinEmoji = "🦅"
	}

	resultText := fmt.Sprintf("%s %s", coinEmoji, strings.ToUpper(string(result.Side)))

	if result.Won {
		profit := result.Payout - result.Bet.Amount
		ui.resultLabel.SetText(fmt.Sprintf("🎉 %s - You won $%.2f! (Profit: +$%.2f)",
			resultText, result.Payout, profit))

		// Show celebration notification
		fyne.CurrentApp().SendNotification(&fyne.Notification{
			Title:   "You Won!",
			Content: fmt.Sprintf("Congratulations! You won $%.2f", result.Payout),
		})
	} else {
		ui.resultLabel.SetText(fmt.Sprintf("😞 %s - You lost $%.2f. Better luck next time!",
			resultText, result.Bet.Amount))
	}
}

// addToHistory adds a result to the game history
func (ui *GameUI) addToHistory(result *game.Result) {
	// Add to beginning of slice (most recent first)
	ui.gameHistory = append([]*game.Result{result}, ui.gameHistory...)

	// Keep only last 50 games for performance
	if len(ui.gameHistory) > 50 {
		ui.gameHistory = ui.gameHistory[:50]
	}

	ui.historyList.Refresh()
}

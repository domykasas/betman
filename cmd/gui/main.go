// Package main provides the GUI interface for the coin flip game using Fyne.
package main

import (
	"context"
	"fmt"
	"os"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/theme"

	"coinflip-game/cmd/gui/ui"
	"coinflip-game/internal/config"
	"coinflip-game/internal/game"
	"coinflip-game/internal/logger"
	"coinflip-game/internal/storage"
)

func main() {
	// Load configuration
	cfg, err := config.Load("")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Initialize logger (use no-op logger for GUI to avoid console spam)
	log := logger.NewNop()

	// Initialize game dependencies
	repo := storage.NewMemoryRepository()
	rng := game.NewDefaultRandomGenerator()
	engine := game.NewEngine(cfg.ToGameConfig(), repo, rng, log)

	// Create Fyne application
	myApp := app.New()
	myApp.SetIcon(nil) // You can set a custom icon here

	// Set theme based on configuration
	// Note: Using deprecated themes for educational purposes
	// In production, consider implementing custom themes
	if cfg.UI.Theme == "light" {
		myApp.Settings().SetTheme(theme.LightTheme())
	} else {
		myApp.Settings().SetTheme(theme.DarkTheme())
	}

	// Create the main window
	ctx := context.Background()
	gameUI := ui.NewGameUI(ctx, myApp, engine, cfg, log)

	// Set window properties
	window := gameUI.GetWindow()
	window.Resize(fyne.NewSize(float32(cfg.UI.WindowWidth), float32(cfg.UI.WindowHeight)))
	window.CenterOnScreen()

	// Show and run the application
	window.ShowAndRun()
}

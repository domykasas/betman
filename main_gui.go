//go:build gui

// main_gui.go is the GUI entry point for the multiplayer coin flip game.
// This provides a unified GUI that supports both single-player and multiplayer modes.
package main

import (
	"context"
	"fmt"
	"os"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/theme"
	"go.uber.org/zap"

	"coinflip-game/cmd/gui/ui"
	"coinflip-game/internal/config"
	"coinflip-game/internal/logger"
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

	// Create Fyne application
	myApp := app.New()
	myApp.SetIcon(nil)

	// Set theme based on configuration
	// Note: Using deprecated themes for educational purposes
	// In production, consider implementing custom themes
	if cfg.UI.Theme == "light" {
		myApp.Settings().SetTheme(theme.LightTheme())
	} else {
		myApp.Settings().SetTheme(theme.DarkTheme())
	}

	// Create the multiplayer game UI (which supports both single and multiplayer modes)
	ctx := context.Background()
	gameUI := ui.NewMultiplayerGameUI(ctx, myApp, cfg, log)

	// Set window properties
	window := gameUI.GetWindow()
	window.Resize(fyne.NewSize(float32(cfg.UI.WindowWidth), float32(cfg.UI.WindowHeight)))
	window.CenterOnScreen()

	log.Info("Starting coin flip game",
		zap.String("mode", "GUI"),
		zap.String("server", fmt.Sprintf("%s:%d", cfg.Multiplayer.ServerHost, cfg.Multiplayer.ServerPort)),
	)

	// Show and run the application
	window.ShowAndRun()
}
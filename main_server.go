//go:build server

// main_server.go is the server entry point for the multiplayer coin flip game.
// This handles WebSocket connections, room management, and synchronized gameplay.
package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/zap"

	"coinflip-game/internal/config"
	"coinflip-game/internal/logger"
	"coinflip-game/internal/network"
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

	// Create server configuration from app config with defaults
	serverConfig := network.DefaultServerConfig()
	if cfg.Multiplayer.ServerHost != "" {
		serverConfig.Host = cfg.Multiplayer.ServerHost
	}
	if cfg.Multiplayer.ServerPort > 0 {
		serverConfig.Port = cfg.Multiplayer.ServerPort
	}
	if cfg.Multiplayer.MaxRooms > 0 {
		serverConfig.MaxRooms = cfg.Multiplayer.MaxRooms
	}
	if cfg.Multiplayer.MaxPlayers > 0 {
		serverConfig.MaxClientsRoom = cfg.Multiplayer.MaxPlayers
	}

	// Create and start the multiplayer server
	server := network.NewServer(serverConfig, log)

	// Handle graceful shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-c
		log.Info("Shutting down server...")
		server.Stop()
		os.Exit(0)
	}()

	log.Info("Starting multiplayer coin flip server",
		zap.String("host", serverConfig.Host),
		zap.Int("port", serverConfig.Port),
		zap.Int("max_rooms", serverConfig.MaxRooms),
		zap.Int("max_players_per_room", serverConfig.MaxClientsRoom),
	)

	// Start the server (this blocks)
	if err := server.Start(); err != nil {
		log.Error("Server failed to start", zap.Error(err))
		os.Exit(1)
	}
}
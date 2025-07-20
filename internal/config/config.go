// Package config provides configuration management for the coin flip game.
// It uses Viper for flexible configuration loading from various sources.
package config

import (
	"fmt"
	"strings"

	"coinflip-game/internal/game"

	"github.com/spf13/viper"
)

// Config represents the complete application configuration
type Config struct {
	Game        GameConfig        `mapstructure:"game"`
	Logging     LoggingConfig     `mapstructure:"logging"`
	UI          UIConfig          `mapstructure:"ui"`
	Multiplayer MultiplayerConfig `mapstructure:"multiplayer"`
}

// GameConfig holds game-specific configuration
type GameConfig struct {
	StartingBalance float64 `mapstructure:"starting_balance"`
	MinBet          float64 `mapstructure:"min_bet"`
	MaxBet          float64 `mapstructure:"max_bet"`
	PayoutRatio     float64 `mapstructure:"payout_ratio"`
}

// LoggingConfig holds logging configuration
type LoggingConfig struct {
	Level       string `mapstructure:"level"`
	Development bool   `mapstructure:"development"`
}

// UIConfig holds user interface configuration
type UIConfig struct {
	Theme        string `mapstructure:"theme"`
	WindowWidth  int    `mapstructure:"window_width"`
	WindowHeight int    `mapstructure:"window_height"`
}

// MultiplayerConfig holds multiplayer server configuration
type MultiplayerConfig struct {
	ServerHost      string `mapstructure:"server_host"`
	ServerPort      int    `mapstructure:"server_port"`
	MaxRooms        int    `mapstructure:"max_rooms"`
	MaxPlayers      int    `mapstructure:"max_players"`
	MinPlayers      int    `mapstructure:"min_players"`
	BettingDuration int    `mapstructure:"betting_duration_seconds"`
	AutoJoin        bool   `mapstructure:"auto_join"`
	DefaultRoom     string `mapstructure:"default_room"`
}

// DefaultConfig returns a configuration with sensible defaults
func DefaultConfig() *Config {
	return &Config{
		Game: GameConfig{
			StartingBalance: 1000.0,
			MinBet:          1.0,
			MaxBet:          100.0,
			PayoutRatio:     2.0,
		},
		Logging: LoggingConfig{
			Level:       "info",
			Development: false,
		},
		UI: UIConfig{
			Theme:        "dark",
			WindowWidth:  800,
			WindowHeight: 600,
		},
		Multiplayer: MultiplayerConfig{
			ServerHost:      "localhost",
			ServerPort:      8080,
			MaxRooms:        100,
			MaxPlayers:      8,
			MinPlayers:      2,
			BettingDuration: 60,
			AutoJoin:        true,
			DefaultRoom:     "lobby",
		},
	}
}

// Load loads configuration from various sources with the following priority:
// 1. Command line flags
// 2. Environment variables
// 3. Configuration file
// 4. Default values
func Load(configPath string) (*Config, error) {
	// Set up Viper
	v := viper.New()

	// Set default values
	setDefaults(v)

	// Configure file reading
	if configPath != "" {
		v.SetConfigFile(configPath)
	} else {
		v.SetConfigName("config")
		v.SetConfigType("json")
		v.AddConfigPath(".")
		v.AddConfigPath("./configs")
		v.AddConfigPath("$HOME/.coinflip")
		v.AddConfigPath("/etc/coinflip")
	}

	// Configure environment variables
	v.SetEnvPrefix("COINFLIP")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// Read configuration file if it exists
	if err := v.ReadInConfig(); err != nil {
		// Don't treat missing config file as an error, just use defaults
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
	}

	// Unmarshal configuration
	var config Config
	if err := v.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &config, nil
}

// setDefaults sets default values in Viper
func setDefaults(v *viper.Viper) {
	defaults := DefaultConfig()

	// Game defaults
	v.SetDefault("game.starting_balance", defaults.Game.StartingBalance)
	v.SetDefault("game.min_bet", defaults.Game.MinBet)
	v.SetDefault("game.max_bet", defaults.Game.MaxBet)
	v.SetDefault("game.payout_ratio", defaults.Game.PayoutRatio)

	// Logging defaults
	v.SetDefault("logging.level", defaults.Logging.Level)
	v.SetDefault("logging.development", defaults.Logging.Development)

	// UI defaults
	v.SetDefault("ui.theme", defaults.UI.Theme)
	v.SetDefault("ui.window_width", defaults.UI.WindowWidth)
	v.SetDefault("ui.window_height", defaults.UI.WindowHeight)

	// Multiplayer defaults
	v.SetDefault("multiplayer.server_host", defaults.Multiplayer.ServerHost)
	v.SetDefault("multiplayer.server_port", defaults.Multiplayer.ServerPort)
	v.SetDefault("multiplayer.max_rooms", defaults.Multiplayer.MaxRooms)
	v.SetDefault("multiplayer.max_players", defaults.Multiplayer.MaxPlayers)
	v.SetDefault("multiplayer.min_players", defaults.Multiplayer.MinPlayers)
	v.SetDefault("multiplayer.betting_duration_seconds", defaults.Multiplayer.BettingDuration)
	v.SetDefault("multiplayer.auto_join", defaults.Multiplayer.AutoJoin)
	v.SetDefault("multiplayer.default_room", defaults.Multiplayer.DefaultRoom)
}

// Validate checks if the configuration values are valid
func (c *Config) Validate() error {
	// Validate game configuration
	if c.Game.StartingBalance <= 0 {
		return fmt.Errorf("starting_balance must be positive, got %f", c.Game.StartingBalance)
	}

	if c.Game.MinBet <= 0 {
		return fmt.Errorf("min_bet must be positive, got %f", c.Game.MinBet)
	}

	if c.Game.MaxBet <= c.Game.MinBet {
		return fmt.Errorf("max_bet (%f) must be greater than min_bet (%f)", c.Game.MaxBet, c.Game.MinBet)
	}

	if c.Game.PayoutRatio <= 1.0 {
		return fmt.Errorf("payout_ratio must be greater than 1.0, got %f", c.Game.PayoutRatio)
	}

	// Validate logging configuration
	validLevels := []string{"debug", "info", "warn", "error", "fatal"}
	levelValid := false
	for _, level := range validLevels {
		if c.Logging.Level == level {
			levelValid = true
			break
		}
	}
	if !levelValid {
		return fmt.Errorf("invalid logging level '%s', must be one of: %v", c.Logging.Level, validLevels)
	}

	// Validate UI configuration
	if c.UI.WindowWidth <= 0 || c.UI.WindowHeight <= 0 {
		return fmt.Errorf("window dimensions must be positive, got %dx%d", c.UI.WindowWidth, c.UI.WindowHeight)
	}

	validThemes := []string{"light", "dark"}
	themeValid := false
	for _, theme := range validThemes {
		if c.UI.Theme == theme {
			themeValid = true
			break
		}
	}
	if !themeValid {
		return fmt.Errorf("invalid theme '%s', must be one of: %v", c.UI.Theme, validThemes)
	}

	return nil
}

// ToGameConfig converts the configuration to a game.Config
func (c *Config) ToGameConfig() game.Config {
	return game.Config{
		StartingBalance: c.Game.StartingBalance,
		MinBet:          c.Game.MinBet,
		MaxBet:          c.Game.MaxBet,
		PayoutRatio:     c.Game.PayoutRatio,
	}
}

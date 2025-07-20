package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	assert.NotNil(t, config)
	assert.Equal(t, 1000.0, config.Game.StartingBalance)
	assert.Equal(t, 1.0, config.Game.MinBet)
	assert.Equal(t, 100.0, config.Game.MaxBet)
	assert.Equal(t, 2.0, config.Game.PayoutRatio)
	assert.Equal(t, "info", config.Logging.Level)
	assert.False(t, config.Logging.Development)
	assert.Equal(t, "dark", config.UI.Theme)
	assert.Equal(t, 800, config.UI.WindowWidth)
	assert.Equal(t, 600, config.UI.WindowHeight)
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name          string
		config        *Config
		expectedError string
	}{
		{
			name:   "valid config",
			config: DefaultConfig(),
		},
		{
			name: "negative starting balance",
			config: &Config{
				Game: GameConfig{
					StartingBalance: -100,
					MinBet:          1,
					MaxBet:          100,
					PayoutRatio:     2.0,
				},
				Logging: LoggingConfig{Level: "info"},
				UI:      UIConfig{Theme: "dark", WindowWidth: 800, WindowHeight: 600},
			},
			expectedError: "starting_balance must be positive",
		},
		{
			name: "zero starting balance",
			config: &Config{
				Game: GameConfig{
					StartingBalance: 0,
					MinBet:          1,
					MaxBet:          100,
					PayoutRatio:     2.0,
				},
				Logging: LoggingConfig{Level: "info"},
				UI:      UIConfig{Theme: "dark", WindowWidth: 800, WindowHeight: 600},
			},
			expectedError: "starting_balance must be positive",
		},
		{
			name: "negative min bet",
			config: &Config{
				Game: GameConfig{
					StartingBalance: 1000,
					MinBet:          -1,
					MaxBet:          100,
					PayoutRatio:     2.0,
				},
				Logging: LoggingConfig{Level: "info"},
				UI:      UIConfig{Theme: "dark", WindowWidth: 800, WindowHeight: 600},
			},
			expectedError: "min_bet must be positive",
		},
		{
			name: "max bet less than min bet",
			config: &Config{
				Game: GameConfig{
					StartingBalance: 1000,
					MinBet:          100,
					MaxBet:          50,
					PayoutRatio:     2.0,
				},
				Logging: LoggingConfig{Level: "info"},
				UI:      UIConfig{Theme: "dark", WindowWidth: 800, WindowHeight: 600},
			},
			expectedError: "max_bet (50.000000) must be greater than min_bet (100.000000)",
		},
		{
			name: "max bet equal to min bet",
			config: &Config{
				Game: GameConfig{
					StartingBalance: 1000,
					MinBet:          100,
					MaxBet:          100,
					PayoutRatio:     2.0,
				},
				Logging: LoggingConfig{Level: "info"},
				UI:      UIConfig{Theme: "dark", WindowWidth: 800, WindowHeight: 600},
			},
			expectedError: "max_bet (100.000000) must be greater than min_bet (100.000000)",
		},
		{
			name: "payout ratio too low",
			config: &Config{
				Game: GameConfig{
					StartingBalance: 1000,
					MinBet:          1,
					MaxBet:          100,
					PayoutRatio:     1.0,
				},
				Logging: LoggingConfig{Level: "info"},
				UI:      UIConfig{Theme: "dark", WindowWidth: 800, WindowHeight: 600},
			},
			expectedError: "payout_ratio must be greater than 1.0",
		},
		{
			name: "invalid logging level",
			config: &Config{
				Game: GameConfig{
					StartingBalance: 1000,
					MinBet:          1,
					MaxBet:          100,
					PayoutRatio:     2.0,
				},
				Logging: LoggingConfig{Level: "invalid"},
				UI:      UIConfig{Theme: "dark", WindowWidth: 800, WindowHeight: 600},
			},
			expectedError: "invalid logging level 'invalid'",
		},
		{
			name: "negative window width",
			config: &Config{
				Game: GameConfig{
					StartingBalance: 1000,
					MinBet:          1,
					MaxBet:          100,
					PayoutRatio:     2.0,
				},
				Logging: LoggingConfig{Level: "info"},
				UI:      UIConfig{Theme: "dark", WindowWidth: -800, WindowHeight: 600},
			},
			expectedError: "window dimensions must be positive",
		},
		{
			name: "negative window height",
			config: &Config{
				Game: GameConfig{
					StartingBalance: 1000,
					MinBet:          1,
					MaxBet:          100,
					PayoutRatio:     2.0,
				},
				Logging: LoggingConfig{Level: "info"},
				UI:      UIConfig{Theme: "dark", WindowWidth: 800, WindowHeight: -600},
			},
			expectedError: "window dimensions must be positive",
		},
		{
			name: "invalid theme",
			config: &Config{
				Game: GameConfig{
					StartingBalance: 1000,
					MinBet:          1,
					MaxBet:          100,
					PayoutRatio:     2.0,
				},
				Logging: LoggingConfig{Level: "info"},
				UI:      UIConfig{Theme: "invalid", WindowWidth: 800, WindowHeight: 600},
			},
			expectedError: "invalid theme 'invalid'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestConfig_ToGameConfig(t *testing.T) {
	config := &Config{
		Game: GameConfig{
			StartingBalance: 500.0,
			MinBet:          5.0,
			MaxBet:          50.0,
			PayoutRatio:     1.5,
		},
	}

	gameConfig := config.ToGameConfig()

	assert.Equal(t, 500.0, gameConfig.StartingBalance)
	assert.Equal(t, 5.0, gameConfig.MinBet)
	assert.Equal(t, 50.0, gameConfig.MaxBet)
	assert.Equal(t, 1.5, gameConfig.PayoutRatio)
}

func TestLoad_DefaultsOnly(t *testing.T) {
	// Load without config file should use defaults
	config, err := Load("")

	assert.NoError(t, err)
	assert.NotNil(t, config)

	// Should match default config
	defaultConfig := DefaultConfig()
	assert.Equal(t, defaultConfig.Game, config.Game)
	assert.Equal(t, defaultConfig.Logging, config.Logging)
	assert.Equal(t, defaultConfig.UI, config.UI)
}

func TestLoad_WithConfigFile(t *testing.T) {
	// Create temporary config file
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "test_config.json")

	configContent := `{
		"game": {
			"starting_balance": 2000.0,
			"min_bet": 5.0,
			"max_bet": 200.0,
			"payout_ratio": 3.0
		},
		"logging": {
			"level": "debug",
			"development": true
		},
		"ui": {
			"theme": "light",
			"window_width": 1024,
			"window_height": 768
		}
	}`

	err := os.WriteFile(configFile, []byte(configContent), 0644)
	require.NoError(t, err)

	// Load config from file
	config, err := Load(configFile)

	assert.NoError(t, err)
	assert.NotNil(t, config)

	// Verify loaded values
	assert.Equal(t, 2000.0, config.Game.StartingBalance)
	assert.Equal(t, 5.0, config.Game.MinBet)
	assert.Equal(t, 200.0, config.Game.MaxBet)
	assert.Equal(t, 3.0, config.Game.PayoutRatio)
	assert.Equal(t, "debug", config.Logging.Level)
	assert.True(t, config.Logging.Development)
	assert.Equal(t, "light", config.UI.Theme)
	assert.Equal(t, 1024, config.UI.WindowWidth)
	assert.Equal(t, 768, config.UI.WindowHeight)
}

func TestLoad_InvalidConfigFile(t *testing.T) {
	// Create temporary invalid config file
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "invalid_config.json")

	invalidContent := `{
		"game": {
			"starting_balance": -100
		}
	}`

	err := os.WriteFile(configFile, []byte(invalidContent), 0644)
	require.NoError(t, err)

	// Load config should fail validation
	config, err := Load(configFile)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid configuration")
	assert.Nil(t, config)
}

func TestLoad_MalformedJSON(t *testing.T) {
	// Create temporary malformed config file
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "malformed_config.json")

	malformedContent := `{
		"game": {
			"starting_balance": 1000
		// missing closing brace
	`

	err := os.WriteFile(configFile, []byte(malformedContent), 0644)
	require.NoError(t, err)

	// Load config should fail parsing
	config, err := Load(configFile)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read config file")
	assert.Nil(t, config)
}

func TestLoad_WithEnvironmentVariables(t *testing.T) {
	// Set environment variables
	os.Setenv("COINFLIP_GAME_STARTING_BALANCE", "1500")
	os.Setenv("COINFLIP_GAME_MIN_BET", "2")
	os.Setenv("COINFLIP_LOGGING_LEVEL", "warn")
	os.Setenv("COINFLIP_UI_THEME", "light")

	// Clean up environment variables after test
	defer func() {
		os.Unsetenv("COINFLIP_GAME_STARTING_BALANCE")
		os.Unsetenv("COINFLIP_GAME_MIN_BET")
		os.Unsetenv("COINFLIP_LOGGING_LEVEL")
		os.Unsetenv("COINFLIP_UI_THEME")
	}()

	// Load config (should pick up environment variables)
	config, err := Load("")

	assert.NoError(t, err)
	assert.NotNil(t, config)

	// Environment variables should override defaults
	assert.Equal(t, 1500.0, config.Game.StartingBalance)
	assert.Equal(t, 2.0, config.Game.MinBet)
	assert.Equal(t, "warn", config.Logging.Level)
	assert.Equal(t, "light", config.UI.Theme)

	// Other values should be defaults
	assert.Equal(t, 100.0, config.Game.MaxBet)
	assert.Equal(t, 2.0, config.Game.PayoutRatio)
}

func TestLoad_FileAndEnvironmentPriority(t *testing.T) {
	// Create temporary config file
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "test_config.json")

	configContent := `{
		"game": {
			"starting_balance": 2000.0,
			"min_bet": 5.0
		},
		"logging": {
			"level": "info"
		}
	}`

	err := os.WriteFile(configFile, []byte(configContent), 0644)
	require.NoError(t, err)

	// Set environment variable that should override file
	os.Setenv("COINFLIP_GAME_STARTING_BALANCE", "3000")
	os.Setenv("COINFLIP_LOGGING_LEVEL", "error")

	defer func() {
		os.Unsetenv("COINFLIP_GAME_STARTING_BALANCE")
		os.Unsetenv("COINFLIP_LOGGING_LEVEL")
	}()

	// Load config
	config, err := Load(configFile)

	assert.NoError(t, err)
	assert.NotNil(t, config)

	// Environment variables should override file values
	assert.Equal(t, 3000.0, config.Game.StartingBalance)
	assert.Equal(t, "error", config.Logging.Level)

	// File values should override defaults where not overridden by env
	assert.Equal(t, 5.0, config.Game.MinBet)

	// Default values for unspecified settings
	assert.Equal(t, 100.0, config.Game.MaxBet)
}

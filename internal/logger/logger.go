// Package logger provides centralized logging configuration for the application.
package logger

import (
	"fmt"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// New creates a new zap logger with the specified configuration
func New(level string, development bool) (*zap.Logger, error) {
	// Parse log level
	zapLevel, err := zapcore.ParseLevel(level)
	if err != nil {
		return nil, fmt.Errorf("invalid log level '%s': %w", level, err)
	}

	var config zap.Config

	if development {
		// Development configuration: human-readable output
		config = zap.NewDevelopmentConfig()
		config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	} else {
		// Production configuration: JSON output
		config = zap.NewProductionConfig()
	}

	// Set the log level
	config.Level = zap.NewAtomicLevelAt(zapLevel)

	// Build the logger
	logger, err := config.Build()
	if err != nil {
		return nil, fmt.Errorf("failed to build logger: %w", err)
	}

	return logger, nil
}

// NewNop creates a no-op logger that discards all log messages (useful for testing)
func NewNop() *zap.Logger {
	return zap.NewNop()
}

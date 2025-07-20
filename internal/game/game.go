// Package game provides the core game logic for the coin flip betting game.
// This package is designed to be UI-agnostic and testable, following clean architecture principles.
package game

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"time"

	"go.uber.org/zap"
)

// Common errors returned by the game engine
var (
	ErrInsufficientBalance = errors.New("insufficient balance for bet")
	ErrInvalidBetAmount    = errors.New("invalid bet amount")
	ErrGameNotActive       = errors.New("game is not active")
	ErrInvalidChoice       = errors.New("invalid choice, must be heads or tails")
)

// Side represents the side of a coin
type Side string

const (
	Heads Side = "heads"
	Tails Side = "tails"
)

// String returns the string representation of the side
func (s Side) String() string {
	return string(s)
}

// IsValid checks if the side is valid
func (s Side) IsValid() bool {
	return s == Heads || s == Tails
}

// Bet represents a single bet placed by a player
type Bet struct {
	ID        string    `json:"id"`
	Amount    float64   `json:"amount"`
	Choice    Side      `json:"choice"`
	Timestamp time.Time `json:"timestamp"`
}

// Result represents the outcome of a coin flip game
type Result struct {
	ID        string    `json:"id"`
	Side      Side      `json:"side"`
	Bet       *Bet      `json:"bet,omitempty"`
	Won       bool      `json:"won"`
	Payout    float64   `json:"payout"`
	Timestamp time.Time `json:"timestamp"`
	Seed      string    `json:"seed"`
}

// Stats represents player statistics
type Stats struct {
	GamesPlayed   int     `json:"games_played"`
	GamesWon      int     `json:"games_won"`
	TotalWagered  float64 `json:"total_wagered"`
	TotalWinnings float64 `json:"total_winnings"`
	NetProfit     float64 `json:"net_profit"`
	WinRate       float64 `json:"win_rate"`
}

// Config holds game configuration
type Config struct {
	StartingBalance float64 `json:"starting_balance"`
	MinBet          float64 `json:"min_bet"`
	MaxBet          float64 `json:"max_bet"`
	PayoutRatio     float64 `json:"payout_ratio"`
}

// Player represents a game player with their current state
type Player struct {
	ID      string  `json:"id"`
	Balance float64 `json:"balance"`
	Stats   Stats   `json:"stats"`
}

// Repository interface for persisting game data
// This allows for dependency injection and easy testing
type Repository interface {
	SaveResult(ctx context.Context, result *Result) error
	GetResults(ctx context.Context, limit int) ([]*Result, error)
	GetStats(ctx context.Context, playerID string) (*Stats, error)
	SavePlayer(ctx context.Context, player *Player) error
	GetPlayer(ctx context.Context, playerID string) (*Player, error)
}

// RandomGenerator interface for generating random numbers
// This allows for deterministic testing by injecting a mock
type RandomGenerator interface {
	GenerateSecureSeed() (string, error)
	FlipCoin(seed string) (Side, error)
}

// Engine is the main game engine that orchestrates coin flip games
type Engine struct {
	config     Config
	repo       Repository
	rng        RandomGenerator
	logger     *zap.Logger
	currentBet *Bet
}

// NewEngine creates a new game engine with the provided dependencies
func NewEngine(config Config, repo Repository, rng RandomGenerator, logger *zap.Logger) *Engine {
	return &Engine{
		config: config,
		repo:   repo,
		rng:    rng,
		logger: logger,
	}
}

// GetConfig returns the current game configuration
func (e *Engine) GetConfig() Config {
	return e.config
}

// CreatePlayer creates a new player with starting balance
func (e *Engine) CreatePlayer(ctx context.Context, playerID string) (*Player, error) {
	player := &Player{
		ID:      playerID,
		Balance: e.config.StartingBalance,
		Stats:   Stats{},
	}

	if err := e.repo.SavePlayer(ctx, player); err != nil {
		e.logger.Error("Failed to save new player", zap.String("player_id", playerID), zap.Error(err))
		return nil, fmt.Errorf("failed to save player: %w", err)
	}

	e.logger.Info("Created new player", zap.String("player_id", playerID), zap.Float64("starting_balance", e.config.StartingBalance))
	return player, nil
}

// GetPlayer retrieves a player by ID, creating one if it doesn't exist
func (e *Engine) GetPlayer(ctx context.Context, playerID string) (*Player, error) {
	player, err := e.repo.GetPlayer(ctx, playerID)
	if err != nil {
		e.logger.Info("Player not found, creating new player", zap.String("player_id", playerID))
		return e.CreatePlayer(ctx, playerID)
	}
	return player, nil
}

// PlaceBet validates and places a bet for the current game round
func (e *Engine) PlaceBet(ctx context.Context, playerID string, amount float64, choice Side) (*Bet, error) {
	// Validate input parameters
	if !choice.IsValid() {
		return nil, ErrInvalidChoice
	}

	if amount < e.config.MinBet || amount > e.config.MaxBet {
		return nil, ErrInvalidBetAmount
	}

	// Get player and validate balance
	player, err := e.GetPlayer(ctx, playerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get player: %w", err)
	}

	if player.Balance < amount {
		return nil, ErrInsufficientBalance
	}

	// Create the bet
	bet := &Bet{
		ID:        e.generateBetID(),
		Amount:    amount,
		Choice:    choice,
		Timestamp: time.Now(),
	}

	// Deduct amount from player balance
	player.Balance -= amount
	if err := e.repo.SavePlayer(ctx, player); err != nil {
		return nil, fmt.Errorf("failed to update player balance: %w", err)
	}

	e.currentBet = bet
	e.logger.Info("Bet placed",
		zap.String("player_id", playerID),
		zap.String("bet_id", bet.ID),
		zap.Float64("amount", amount),
		zap.String("choice", choice.String()),
	)

	return bet, nil
}

// FlipCoin executes the coin flip and determines the result
func (e *Engine) FlipCoin(ctx context.Context, playerID string) (*Result, error) {
	if e.currentBet == nil {
		return nil, ErrGameNotActive
	}

	// Generate secure random seed for the coin flip
	seed, err := e.rng.GenerateSecureSeed()
	if err != nil {
		return nil, fmt.Errorf("failed to generate random seed: %w", err)
	}

	// Flip the coin using the seed
	coinSide, err := e.rng.FlipCoin(seed)
	if err != nil {
		return nil, fmt.Errorf("failed to flip coin: %w", err)
	}

	// Determine if the bet won
	won := e.currentBet.Choice == coinSide
	var payout float64
	if won {
		payout = e.currentBet.Amount * e.config.PayoutRatio
	}

	// Create the result
	result := &Result{
		ID:        e.generateResultID(),
		Side:      coinSide,
		Bet:       e.currentBet,
		Won:       won,
		Payout:    payout,
		Timestamp: time.Now(),
		Seed:      seed,
	}

	// Update player balance and stats
	player, err := e.GetPlayer(ctx, playerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get player for result processing: %w", err)
	}

	// Add payout to balance if won
	if won {
		player.Balance += payout
	}

	// Update statistics
	player.Stats.GamesPlayed++
	player.Stats.TotalWagered += e.currentBet.Amount
	if won {
		player.Stats.GamesWon++
		player.Stats.TotalWinnings += payout
	}
	player.Stats.NetProfit = player.Stats.TotalWinnings - player.Stats.TotalWagered
	if player.Stats.GamesPlayed > 0 {
		player.Stats.WinRate = float64(player.Stats.GamesWon) / float64(player.Stats.GamesPlayed) * 100
	}

	// Save updated player data
	if err := e.repo.SavePlayer(ctx, player); err != nil {
		e.logger.Error("Failed to save player after game", zap.String("player_id", playerID), zap.Error(err))
		return nil, fmt.Errorf("failed to save player: %w", err)
	}

	// Save the result
	if err := e.repo.SaveResult(ctx, result); err != nil {
		e.logger.Error("Failed to save game result", zap.String("result_id", result.ID), zap.Error(err))
		return nil, fmt.Errorf("failed to save result: %w", err)
	}

	// Clear current bet
	e.currentBet = nil

	e.logger.Info("Game completed",
		zap.String("player_id", playerID),
		zap.String("result_id", result.ID),
		zap.String("coin_side", coinSide.String()),
		zap.Bool("won", won),
		zap.Float64("payout", payout),
	)

	return result, nil
}

// GetGameHistory returns the recent game results
func (e *Engine) GetGameHistory(ctx context.Context, limit int) ([]*Result, error) {
	return e.repo.GetResults(ctx, limit)
}

// GetCurrentBet returns the current active bet, if any
func (e *Engine) GetCurrentBet() *Bet {
	return e.currentBet
}

// CancelCurrentBet cancels the current bet and refunds the player
func (e *Engine) CancelCurrentBet(ctx context.Context, playerID string) error {
	if e.currentBet == nil {
		return ErrGameNotActive
	}

	// Refund the bet amount to player
	player, err := e.GetPlayer(ctx, playerID)
	if err != nil {
		return fmt.Errorf("failed to get player for refund: %w", err)
	}

	player.Balance += e.currentBet.Amount
	if err := e.repo.SavePlayer(ctx, player); err != nil {
		return fmt.Errorf("failed to refund player: %w", err)
	}

	e.logger.Info("Bet cancelled and refunded",
		zap.String("player_id", playerID),
		zap.String("bet_id", e.currentBet.ID),
		zap.Float64("refund_amount", e.currentBet.Amount),
	)

	e.currentBet = nil
	return nil
}

// generateBetID creates a unique identifier for a bet
func (e *Engine) generateBetID() string {
	timestamp := time.Now().UnixNano()
	return fmt.Sprintf("bet_%d", timestamp)
}

// generateResultID creates a unique identifier for a game result
func (e *Engine) generateResultID() string {
	timestamp := time.Now().UnixNano()
	return fmt.Sprintf("result_%d", timestamp)
}

// DefaultRandomGenerator implements RandomGenerator using crypto/rand
type DefaultRandomGenerator struct{}

// NewDefaultRandomGenerator creates a new DefaultRandomGenerator
func NewDefaultRandomGenerator() *DefaultRandomGenerator {
	return &DefaultRandomGenerator{}
}

// GenerateSecureSeed generates a cryptographically secure random seed
func (rng *DefaultRandomGenerator) GenerateSecureSeed() (string, error) {
	seedBytes := make([]byte, 32)
	if _, err := rand.Read(seedBytes); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}

	hash := sha256.Sum256(seedBytes)
	return fmt.Sprintf("%x", hash), nil
}

// FlipCoin uses the provided seed to deterministically flip a coin
func (rng *DefaultRandomGenerator) FlipCoin(seed string) (Side, error) {
	if seed == "" {
		return "", errors.New("seed cannot be empty")
	}

	// Hash the seed to get deterministic randomness
	hash := sha256.Sum256([]byte(seed))

	// Use the first 8 bytes to get a uint64
	randomValue := binary.BigEndian.Uint64(hash[:8])

	// Even numbers = heads, odd numbers = tails
	if randomValue%2 == 0 {
		return Heads, nil
	}
	return Tails, nil
}

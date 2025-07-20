package game

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap/zaptest"
)

// MockRepository implements the Repository interface for testing
type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) SaveResult(ctx context.Context, result *Result) error {
	args := m.Called(ctx, result)
	return args.Error(0)
}

func (m *MockRepository) GetResults(ctx context.Context, limit int) ([]*Result, error) {
	args := m.Called(ctx, limit)
	return args.Get(0).([]*Result), args.Error(1)
}

func (m *MockRepository) GetStats(ctx context.Context, playerID string) (*Stats, error) {
	args := m.Called(ctx, playerID)
	return args.Get(0).(*Stats), args.Error(1)
}

func (m *MockRepository) SavePlayer(ctx context.Context, player *Player) error {
	args := m.Called(ctx, player)
	return args.Error(0)
}

func (m *MockRepository) GetPlayer(ctx context.Context, playerID string) (*Player, error) {
	args := m.Called(ctx, playerID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Player), args.Error(1)
}

// MockRandomGenerator implements the RandomGenerator interface for testing
type MockRandomGenerator struct {
	mock.Mock
}

func (m *MockRandomGenerator) GenerateSecureSeed() (string, error) {
	args := m.Called()
	return args.String(0), args.Error(1)
}

func (m *MockRandomGenerator) FlipCoin(seed string) (Side, error) {
	args := m.Called(seed)
	return Side(args.String(0)), args.Error(1)
}

func TestSide_String(t *testing.T) {
	assert.Equal(t, "heads", Heads.String())
	assert.Equal(t, "tails", Tails.String())
}

func TestSide_IsValid(t *testing.T) {
	assert.True(t, Heads.IsValid())
	assert.True(t, Tails.IsValid())
	assert.False(t, Side("invalid").IsValid())
	assert.False(t, Side("").IsValid())
}

func TestNewEngine(t *testing.T) {
	config := Config{
		StartingBalance: 1000,
		MinBet:          1,
		MaxBet:          100,
		PayoutRatio:     2.0,
	}
	repo := &MockRepository{}
	rng := &MockRandomGenerator{}
	logger := zaptest.NewLogger(t)

	engine := NewEngine(config, repo, rng, logger)

	assert.NotNil(t, engine)
	assert.Equal(t, config, engine.GetConfig())
	assert.Nil(t, engine.GetCurrentBet())
}

func TestEngine_CreatePlayer(t *testing.T) {
	tests := []struct {
		name          string
		playerID      string
		saveError     error
		expectedError string
	}{
		{
			name:     "successful creation",
			playerID: "test_player",
		},
		{
			name:          "save error",
			playerID:      "test_player",
			saveError:     errors.New("save failed"),
			expectedError: "failed to save player",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := Config{StartingBalance: 1000, MinBet: 1, MaxBet: 100, PayoutRatio: 2.0}
			repo := &MockRepository{}
			rng := &MockRandomGenerator{}
			logger := zaptest.NewLogger(t)
			engine := NewEngine(config, repo, rng, logger)

			ctx := context.Background()

			// Set up mock expectations
			repo.On("SavePlayer", ctx, mock.MatchedBy(func(p *Player) bool {
				return p.ID == tt.playerID && p.Balance == 1000
			})).Return(tt.saveError)

			player, err := engine.CreatePlayer(ctx, tt.playerID)

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
				assert.Nil(t, player)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, player)
				assert.Equal(t, tt.playerID, player.ID)
				assert.Equal(t, 1000.0, player.Balance)
			}

			repo.AssertExpectations(t)
		})
	}
}

func TestEngine_GetPlayer(t *testing.T) {
	tests := []struct {
		name           string
		playerID       string
		existingPlayer *Player
		getError       error
		saveError      error
		expectedError  string
	}{
		{
			name:     "existing player",
			playerID: "existing_player",
			existingPlayer: &Player{
				ID:      "existing_player",
				Balance: 500,
			},
		},
		{
			name:      "new player creation",
			playerID:  "new_player",
			getError:  errors.New("player not found"),
			saveError: nil,
		},
		{
			name:          "creation fails",
			playerID:      "new_player",
			getError:      errors.New("player not found"),
			saveError:     errors.New("save failed"),
			expectedError: "failed to save player",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := Config{StartingBalance: 1000, MinBet: 1, MaxBet: 100, PayoutRatio: 2.0}
			repo := &MockRepository{}
			rng := &MockRandomGenerator{}
			logger := zaptest.NewLogger(t)
			engine := NewEngine(config, repo, rng, logger)

			ctx := context.Background()

			// Set up mock expectations
			repo.On("GetPlayer", ctx, tt.playerID).Return(tt.existingPlayer, tt.getError)
			if tt.getError != nil {
				repo.On("SavePlayer", ctx, mock.MatchedBy(func(p *Player) bool {
					return p.ID == tt.playerID
				})).Return(tt.saveError)
			}

			player, err := engine.GetPlayer(ctx, tt.playerID)

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, player)
				assert.Equal(t, tt.playerID, player.ID)
			}

			repo.AssertExpectations(t)
		})
	}
}

func TestEngine_PlaceBet(t *testing.T) {
	tests := []struct {
		name          string
		amount        float64
		choice        Side
		playerBalance float64
		existingBet   *Bet
		getError      error
		saveError     error
		expectedError string
	}{
		{
			name:          "successful bet",
			amount:        10,
			choice:        Heads,
			playerBalance: 100,
		},
		{
			name:          "invalid choice",
			amount:        10,
			choice:        Side("invalid"),
			expectedError: "invalid choice",
		},
		{
			name:          "bet too low",
			amount:        0.5,
			choice:        Heads,
			expectedError: "invalid bet amount",
		},
		{
			name:          "bet too high",
			amount:        150,
			choice:        Heads,
			expectedError: "invalid bet amount",
		},
		{
			name:          "insufficient balance",
			amount:        10,
			choice:        Heads,
			playerBalance: 5,
			expectedError: "insufficient balance",
		},
		{
			name:          "save player error",
			amount:        10,
			choice:        Heads,
			playerBalance: 100,
			saveError:     errors.New("save failed"),
			expectedError: "failed to update player balance",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := Config{StartingBalance: 1000, MinBet: 1, MaxBet: 100, PayoutRatio: 2.0}
			repo := &MockRepository{}
			rng := &MockRandomGenerator{}
			logger := zaptest.NewLogger(t)
			engine := NewEngine(config, repo, rng, logger)

			ctx := context.Background()
			playerID := "test_player"

			// Set up existing bet if specified
			if tt.existingBet != nil {
				engine.currentBet = tt.existingBet
			}

			// Set up mock expectations
			if tt.getError == nil && tt.choice.IsValid() && tt.amount >= 1 && tt.amount <= 100 {
				player := &Player{
					ID:      playerID,
					Balance: tt.playerBalance,
				}
				repo.On("GetPlayer", ctx, playerID).Return(player, nil)

				if tt.playerBalance >= tt.amount {
					updatedPlayer := &Player{
						ID:      playerID,
						Balance: tt.playerBalance - tt.amount,
					}
					repo.On("SavePlayer", ctx, mock.MatchedBy(func(p *Player) bool {
						return p.Balance == updatedPlayer.Balance
					})).Return(tt.saveError)
				}
			}

			bet, err := engine.PlaceBet(ctx, playerID, tt.amount, tt.choice)

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
				assert.Nil(t, bet)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, bet)
				assert.Equal(t, tt.amount, bet.Amount)
				assert.Equal(t, tt.choice, bet.Choice)
				assert.Equal(t, bet, engine.GetCurrentBet())
			}

			repo.AssertExpectations(t)
		})
	}
}

func TestEngine_FlipCoin(t *testing.T) {
	tests := []struct {
		name            string
		hasBet          bool
		betChoice       Side
		coinResult      Side
		seedGenError    error
		flipError       error
		getPlayerError  error
		savePlayerError error
		saveResultError error
		expectedWin     bool
		expectedError   string
	}{
		{
			name:          "no active bet",
			hasBet:        false,
			expectedError: "game is not active",
		},
		{
			name:          "seed generation error",
			hasBet:        true,
			betChoice:     Heads,
			seedGenError:  errors.New("seed failed"),
			expectedError: "failed to generate random seed",
		},
		{
			name:          "flip error",
			hasBet:        true,
			betChoice:     Heads,
			flipError:     errors.New("flip failed"),
			expectedError: "failed to flip coin",
		},
		{
			name:        "winning bet",
			hasBet:      true,
			betChoice:   Heads,
			coinResult:  Heads,
			expectedWin: true,
		},
		{
			name:        "losing bet",
			hasBet:      true,
			betChoice:   Heads,
			coinResult:  Tails,
			expectedWin: false,
		},
		{
			name:           "get player error",
			hasBet:         true,
			betChoice:      Heads,
			coinResult:     Heads,
			getPlayerError: errors.New("get failed"),
			expectedError:  "failed to get player for result processing",
		},
		{
			name:            "save player error",
			hasBet:          true,
			betChoice:       Heads,
			coinResult:      Heads,
			savePlayerError: errors.New("save failed"),
			expectedError:   "failed to save player",
		},
		{
			name:            "save result error",
			hasBet:          true,
			betChoice:       Heads,
			coinResult:      Heads,
			saveResultError: errors.New("save failed"),
			expectedError:   "failed to save result",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := Config{StartingBalance: 1000, MinBet: 1, MaxBet: 100, PayoutRatio: 2.0}
			repo := &MockRepository{}
			rng := &MockRandomGenerator{}
			logger := zaptest.NewLogger(t)
			engine := NewEngine(config, repo, rng, logger)

			ctx := context.Background()
			playerID := "test_player"

			// Set up current bet if specified
			if tt.hasBet {
				engine.currentBet = &Bet{
					ID:        "test_bet",
					Amount:    10,
					Choice:    tt.betChoice,
					Timestamp: time.Now(),
				}
			}

			// Set up mock expectations
			if tt.hasBet {
				rng.On("GenerateSecureSeed").Return("test_seed", tt.seedGenError)

				if tt.seedGenError == nil {
					// Always set up FlipCoin mock if seed generation succeeds
					rng.On("FlipCoin", "test_seed").Return(string(tt.coinResult), tt.flipError)

					if tt.flipError == nil {
						if tt.getPlayerError == nil {
							player := &Player{
								ID:      playerID,
								Balance: 100,
								Stats:   Stats{},
							}
							repo.On("GetPlayer", ctx, playerID).Return(player, tt.getPlayerError)

							if tt.savePlayerError != nil {
								repo.On("SavePlayer", ctx, mock.AnythingOfType("*game.Player")).Return(tt.savePlayerError)
							} else if tt.saveResultError != nil {
								repo.On("SavePlayer", ctx, mock.AnythingOfType("*game.Player")).Return(nil)
								repo.On("SaveResult", ctx, mock.AnythingOfType("*game.Result")).Return(tt.saveResultError)
							} else {
								repo.On("SavePlayer", ctx, mock.AnythingOfType("*game.Player")).Return(nil)
								repo.On("SaveResult", ctx, mock.AnythingOfType("*game.Result")).Return(nil)
							}
						} else {
							// When GetPlayer fails, engine will try to create a new player
							repo.On("GetPlayer", ctx, playerID).Return(nil, tt.getPlayerError)
							repo.On("SavePlayer", ctx, mock.AnythingOfType("*game.Player")).Return(tt.getPlayerError)
						}
					}
				}
			}

			result, err := engine.FlipCoin(ctx, playerID)

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, tt.coinResult, result.Side)
				assert.Equal(t, tt.expectedWin, result.Won)
				assert.Nil(t, engine.GetCurrentBet()) // Bet should be cleared

				if tt.expectedWin {
					assert.Equal(t, 20.0, result.Payout) // 10 * 2.0 payout ratio
				} else {
					assert.Equal(t, 0.0, result.Payout)
				}
			}

			repo.AssertExpectations(t)
			rng.AssertExpectations(t)
		})
	}
}

func TestEngine_CancelCurrentBet(t *testing.T) {
	tests := []struct {
		name          string
		hasBet        bool
		getError      error
		saveError     error
		expectedError string
	}{
		{
			name:          "no active bet",
			hasBet:        false,
			expectedError: "game is not active",
		},
		{
			name:   "successful cancel",
			hasBet: true,
		},
		{
			name:          "get player error",
			hasBet:        true,
			getError:      errors.New("get failed"),
			expectedError: "failed to get player for refund",
		},
		{
			name:          "save player error",
			hasBet:        true,
			saveError:     errors.New("save failed"),
			expectedError: "failed to refund player",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := Config{StartingBalance: 1000, MinBet: 1, MaxBet: 100, PayoutRatio: 2.0}
			repo := &MockRepository{}
			rng := &MockRandomGenerator{}
			logger := zaptest.NewLogger(t)
			engine := NewEngine(config, repo, rng, logger)

			ctx := context.Background()
			playerID := "test_player"

			// Set up current bet if specified
			if tt.hasBet {
				engine.currentBet = &Bet{
					ID:     "test_bet",
					Amount: 10,
					Choice: Heads,
				}
			}

			// Set up mock expectations
			if tt.hasBet && tt.getError == nil {
				player := &Player{
					ID:      playerID,
					Balance: 90, // Already deducted bet amount
				}
				repo.On("GetPlayer", ctx, playerID).Return(player, tt.getError)
				repo.On("SavePlayer", ctx, mock.MatchedBy(func(p *Player) bool {
					return p.Balance == 100 // Refunded amount
				})).Return(tt.saveError)
			} else if tt.hasBet {
				repo.On("GetPlayer", ctx, playerID).Return(nil, tt.getError)
				// When GetPlayer fails, engine will try to create a new player
				repo.On("SavePlayer", ctx, mock.AnythingOfType("*game.Player")).Return(tt.getError)
			}

			err := engine.CancelCurrentBet(ctx, playerID)

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
				assert.Nil(t, engine.GetCurrentBet())
			}

			repo.AssertExpectations(t)
		})
	}
}

func TestEngine_GetGameHistory(t *testing.T) {
	config := Config{StartingBalance: 1000, MinBet: 1, MaxBet: 100, PayoutRatio: 2.0}
	repo := &MockRepository{}
	rng := &MockRandomGenerator{}
	logger := zaptest.NewLogger(t)
	engine := NewEngine(config, repo, rng, logger)

	ctx := context.Background()
	limit := 10

	expectedResults := []*Result{
		{ID: "1", Side: Heads, Won: true},
		{ID: "2", Side: Tails, Won: false},
	}

	repo.On("GetResults", ctx, limit).Return(expectedResults, nil)

	results, err := engine.GetGameHistory(ctx, limit)

	assert.NoError(t, err)
	assert.Equal(t, expectedResults, results)
	repo.AssertExpectations(t)
}

func TestDefaultRandomGenerator_GenerateSecureSeed(t *testing.T) {
	rng := NewDefaultRandomGenerator()

	seed, err := rng.GenerateSecureSeed()

	assert.NoError(t, err)
	assert.NotEmpty(t, seed)
	assert.Len(t, seed, 64) // SHA256 hex string length
}

func TestDefaultRandomGenerator_FlipCoin(t *testing.T) {
	tests := []struct {
		name          string
		seed          string
		expectedError string
	}{
		{
			name: "valid seed",
			seed: "test_seed_123",
		},
		{
			name:          "empty seed",
			seed:          "",
			expectedError: "seed cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rng := NewDefaultRandomGenerator()

			result, err := rng.FlipCoin(tt.seed)

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
				assert.Empty(t, result)
			} else {
				assert.NoError(t, err)
				assert.True(t, result == Heads || result == Tails)

				// Test deterministic behavior
				result2, err2 := rng.FlipCoin(tt.seed)
				assert.NoError(t, err2)
				assert.Equal(t, result, result2)
			}
		})
	}
}

// Benchmark tests for performance
func BenchmarkDefaultRandomGenerator_GenerateSecureSeed(b *testing.B) {
	rng := NewDefaultRandomGenerator()

	for i := 0; i < b.N; i++ {
		_, err := rng.GenerateSecureSeed()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDefaultRandomGenerator_FlipCoin(b *testing.B) {
	rng := NewDefaultRandomGenerator()
	seed := "benchmark_seed_123"

	for i := 0; i < b.N; i++ {
		_, err := rng.FlipCoin(seed)
		if err != nil {
			b.Fatal(err)
		}
	}
}

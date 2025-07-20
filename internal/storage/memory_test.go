package storage

import (
	"context"
	"fmt"
	"testing"
	"time"

	"coinflip-game/internal/game"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMemoryRepository(t *testing.T) {
	repo := NewMemoryRepository()

	assert.NotNil(t, repo)
	assert.NotNil(t, repo.results)
	assert.NotNil(t, repo.players)
	assert.Equal(t, 0, len(repo.results))
	assert.Equal(t, 0, len(repo.players))
}

func TestMemoryRepository_SaveResult(t *testing.T) {
	tests := []struct {
		name          string
		result        *game.Result
		expectedError string
	}{
		{
			name:          "nil result",
			result:        nil,
			expectedError: "result cannot be nil",
		},
		{
			name: "empty ID",
			result: &game.Result{
				ID:   "",
				Side: game.Heads,
			},
			expectedError: "result ID cannot be empty",
		},
		{
			name: "successful save",
			result: &game.Result{
				ID:        "test_result_1",
				Side:      game.Heads,
				Won:       true,
				Payout:    20.0,
				Timestamp: time.Now(),
				Seed:      "test_seed",
				Bet: &game.Bet{
					ID:        "test_bet_1",
					Amount:    10.0,
					Choice:    game.Heads,
					Timestamp: time.Now(),
				},
			},
		},
		{
			name: "result without bet",
			result: &game.Result{
				ID:        "test_result_2",
				Side:      game.Tails,
				Won:       false,
				Payout:    0.0,
				Timestamp: time.Now(),
				Seed:      "test_seed_2",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := NewMemoryRepository()
			ctx := context.Background()

			err := repo.SaveResult(ctx, tt.result)

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
				assert.Equal(t, 0, repo.GetResultCount())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, 1, repo.GetResultCount())

				// Verify the result was saved correctly
				savedResult := repo.results[tt.result.ID]
				assert.NotNil(t, savedResult)
				assert.Equal(t, tt.result.ID, savedResult.ID)
				assert.Equal(t, tt.result.Side, savedResult.Side)
				assert.Equal(t, tt.result.Won, savedResult.Won)
				assert.Equal(t, tt.result.Payout, savedResult.Payout)
				assert.Equal(t, tt.result.Seed, savedResult.Seed)

				if tt.result.Bet != nil {
					assert.NotNil(t, savedResult.Bet)
					assert.Equal(t, tt.result.Bet.ID, savedResult.Bet.ID)
					assert.Equal(t, tt.result.Bet.Amount, savedResult.Bet.Amount)
					assert.Equal(t, tt.result.Bet.Choice, savedResult.Bet.Choice)
				}
			}
		})
	}
}

func TestMemoryRepository_GetResults(t *testing.T) {
	repo := NewMemoryRepository()
	ctx := context.Background()

	// Test empty repository
	results, err := repo.GetResults(ctx, 10)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(results))

	// Test with zero limit
	results, err = repo.GetResults(ctx, 0)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(results))

	// Add some test results
	now := time.Now()
	testResults := []*game.Result{
		{
			ID:        "result_1",
			Side:      game.Heads,
			Won:       true,
			Timestamp: now.Add(-2 * time.Hour),
		},
		{
			ID:        "result_2",
			Side:      game.Tails,
			Won:       false,
			Timestamp: now.Add(-1 * time.Hour),
		},
		{
			ID:        "result_3",
			Side:      game.Heads,
			Won:       true,
			Timestamp: now,
		},
	}

	for _, result := range testResults {
		err := repo.SaveResult(ctx, result)
		require.NoError(t, err)
	}

	// Test getting all results
	results, err = repo.GetResults(ctx, 10)
	assert.NoError(t, err)
	assert.Equal(t, 3, len(results))

	// Results should be ordered by timestamp descending (most recent first)
	assert.Equal(t, "result_3", results[0].ID) // Most recent
	assert.Equal(t, "result_2", results[1].ID)
	assert.Equal(t, "result_1", results[2].ID) // Oldest

	// Test with limit
	results, err = repo.GetResults(ctx, 2)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(results))
	assert.Equal(t, "result_3", results[0].ID)
	assert.Equal(t, "result_2", results[1].ID)

	// Test limit larger than available results
	results, err = repo.GetResults(ctx, 100)
	assert.NoError(t, err)
	assert.Equal(t, 3, len(results))
}

func TestMemoryRepository_SavePlayer(t *testing.T) {
	tests := []struct {
		name          string
		player        *game.Player
		expectedError string
	}{
		{
			name:          "nil player",
			player:        nil,
			expectedError: "player cannot be nil",
		},
		{
			name: "empty ID",
			player: &game.Player{
				ID:      "",
				Balance: 100,
			},
			expectedError: "player ID cannot be empty",
		},
		{
			name: "successful save",
			player: &game.Player{
				ID:      "test_player_1",
				Balance: 500.0,
				Stats: game.Stats{
					GamesPlayed:   10,
					GamesWon:      6,
					TotalWagered:  100.0,
					TotalWinnings: 120.0,
					NetProfit:     20.0,
					WinRate:       60.0,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := NewMemoryRepository()
			ctx := context.Background()

			err := repo.SavePlayer(ctx, tt.player)

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
				assert.Equal(t, 0, repo.GetPlayerCount())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, 1, repo.GetPlayerCount())

				// Verify the player was saved correctly
				savedPlayer := repo.players[tt.player.ID]
				assert.NotNil(t, savedPlayer)
				assert.Equal(t, tt.player.ID, savedPlayer.ID)
				assert.Equal(t, tt.player.Balance, savedPlayer.Balance)
				assert.Equal(t, tt.player.Stats, savedPlayer.Stats)
			}
		})
	}
}

func TestMemoryRepository_GetPlayer(t *testing.T) {
	repo := NewMemoryRepository()
	ctx := context.Background()

	// Test getting non-existent player
	player, err := repo.GetPlayer(ctx, "non_existent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "player not found")
	assert.Nil(t, player)

	// Test empty player ID
	player, err = repo.GetPlayer(ctx, "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "player ID cannot be empty")
	assert.Nil(t, player)

	// Add a test player
	testPlayer := &game.Player{
		ID:      "test_player",
		Balance: 750.0,
		Stats: game.Stats{
			GamesPlayed: 5,
			GamesWon:    3,
			WinRate:     60.0,
		},
	}

	err = repo.SavePlayer(ctx, testPlayer)
	require.NoError(t, err)

	// Test getting existing player
	player, err = repo.GetPlayer(ctx, "test_player")
	assert.NoError(t, err)
	assert.NotNil(t, player)
	assert.Equal(t, testPlayer.ID, player.ID)
	assert.Equal(t, testPlayer.Balance, player.Balance)
	assert.Equal(t, testPlayer.Stats, player.Stats)

	// Verify it's a copy (modify original and check retrieved is unchanged)
	testPlayer.Balance = 999.0
	player, err = repo.GetPlayer(ctx, "test_player")
	assert.NoError(t, err)
	assert.Equal(t, 750.0, player.Balance) // Should be unchanged
}

func TestMemoryRepository_GetStats(t *testing.T) {
	repo := NewMemoryRepository()
	ctx := context.Background()

	// Test empty player ID
	stats, err := repo.GetStats(ctx, "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "player ID cannot be empty")
	assert.Nil(t, stats)

	// Test getting stats for non-existent player
	stats, err = repo.GetStats(ctx, "non_existent")
	assert.NoError(t, err)
	assert.NotNil(t, stats)
	assert.Equal(t, game.Stats{}, *stats) // Should return empty stats

	// Add a test player with stats
	testPlayer := &game.Player{
		ID:      "test_player",
		Balance: 500.0,
		Stats: game.Stats{
			GamesPlayed:   20,
			GamesWon:      12,
			TotalWagered:  200.0,
			TotalWinnings: 240.0,
			NetProfit:     40.0,
			WinRate:       60.0,
		},
	}

	err = repo.SavePlayer(ctx, testPlayer)
	require.NoError(t, err)

	// Test getting stats for existing player
	stats, err = repo.GetStats(ctx, "test_player")
	assert.NoError(t, err)
	assert.NotNil(t, stats)
	assert.Equal(t, testPlayer.Stats, *stats)

	// Verify it's a copy
	testPlayer.Stats.GamesPlayed = 999
	stats, err = repo.GetStats(ctx, "test_player")
	assert.NoError(t, err)
	assert.Equal(t, 20, stats.GamesPlayed) // Should be unchanged
}

func TestMemoryRepository_Clear(t *testing.T) {
	repo := NewMemoryRepository()
	ctx := context.Background()

	// Add some test data
	testResult := &game.Result{
		ID:   "test_result",
		Side: game.Heads,
	}
	testPlayer := &game.Player{
		ID:      "test_player",
		Balance: 100,
	}

	err := repo.SaveResult(ctx, testResult)
	require.NoError(t, err)
	err = repo.SavePlayer(ctx, testPlayer)
	require.NoError(t, err)

	assert.Equal(t, 1, repo.GetResultCount())
	assert.Equal(t, 1, repo.GetPlayerCount())

	// Clear the repository
	repo.Clear()

	assert.Equal(t, 0, repo.GetResultCount())
	assert.Equal(t, 0, repo.GetPlayerCount())
}

func TestMemoryRepository_ConcurrentAccess(t *testing.T) {
	repo := NewMemoryRepository()
	ctx := context.Background()

	// Test concurrent writes and reads
	const numGoroutines = 10
	const numOperations = 50

	done := make(chan bool, numGoroutines*2)

	// Concurrent writes
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			for j := 0; j < numOperations; j++ {
				result := &game.Result{
					ID:        fmt.Sprintf("result_%d_%d", id, j),
					Side:      game.Heads,
					Timestamp: time.Now(),
				}
				err := repo.SaveResult(ctx, result)
				assert.NoError(t, err)

				player := &game.Player{
					ID:      fmt.Sprintf("player_%d_%d", id, j),
					Balance: float64(id * j),
				}
				err = repo.SavePlayer(ctx, player)
				assert.NoError(t, err)
			}
			done <- true
		}(i)
	}

	// Concurrent reads
	for i := 0; i < numGoroutines; i++ {
		go func() {
			for j := 0; j < numOperations; j++ {
				results, err := repo.GetResults(ctx, 10)
				assert.NoError(t, err)
				assert.True(t, len(results) >= 0)

				stats, err := repo.GetStats(ctx, "non_existent")
				assert.NoError(t, err)
				assert.NotNil(t, stats)
			}
			done <- true
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines*2; i++ {
		<-done
	}

	// Verify final state
	assert.Equal(t, numGoroutines*numOperations, repo.GetResultCount())
	assert.Equal(t, numGoroutines*numOperations, repo.GetPlayerCount())
}

func TestMemoryRepository_DataIntegrity(t *testing.T) {
	repo := NewMemoryRepository()
	ctx := context.Background()

	// Create original data
	originalResult := &game.Result{
		ID:        "test_result",
		Side:      game.Heads,
		Won:       true,
		Payout:    20.0,
		Timestamp: time.Now(),
		Bet: &game.Bet{
			ID:     "test_bet",
			Amount: 10.0,
			Choice: game.Heads,
		},
	}

	originalPlayer := &game.Player{
		ID:      "test_player",
		Balance: 500.0,
		Stats: game.Stats{
			GamesPlayed: 10,
			GamesWon:    6,
		},
	}

	// Save data
	err := repo.SaveResult(ctx, originalResult)
	require.NoError(t, err)
	err = repo.SavePlayer(ctx, originalPlayer)
	require.NoError(t, err)

	// Modify original data
	originalResult.Won = false
	originalResult.Payout = 0.0
	originalResult.Bet.Amount = 999.0
	originalPlayer.Balance = 999.0
	originalPlayer.Stats.GamesPlayed = 999

	// Retrieve data and verify it wasn't affected by modifications
	results, err := repo.GetResults(ctx, 1)
	require.NoError(t, err)
	require.Len(t, results, 1)

	retrievedResult := results[0]
	assert.True(t, retrievedResult.Won)               // Should still be true
	assert.Equal(t, 20.0, retrievedResult.Payout)     // Should still be 20.0
	assert.Equal(t, 10.0, retrievedResult.Bet.Amount) // Should still be 10.0

	retrievedPlayer, err := repo.GetPlayer(ctx, "test_player")
	require.NoError(t, err)
	assert.Equal(t, 500.0, retrievedPlayer.Balance)        // Should still be 500.0
	assert.Equal(t, 10, retrievedPlayer.Stats.GamesPlayed) // Should still be 10
}

// Benchmark tests
func BenchmarkMemoryRepository_SaveResult(b *testing.B) {
	repo := NewMemoryRepository()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result := &game.Result{
			ID:        fmt.Sprintf("result_%d", i),
			Side:      game.Heads,
			Timestamp: time.Now(),
		}
		repo.SaveResult(ctx, result)
	}
}

func BenchmarkMemoryRepository_GetResults(b *testing.B) {
	repo := NewMemoryRepository()
	ctx := context.Background()

	// Prepare test data
	for i := 0; i < 1000; i++ {
		result := &game.Result{
			ID:        fmt.Sprintf("result_%d", i),
			Side:      game.Heads,
			Timestamp: time.Now().Add(time.Duration(i) * time.Second),
		}
		repo.SaveResult(ctx, result)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		repo.GetResults(ctx, 10)
	}
}

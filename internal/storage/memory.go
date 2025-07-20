// Package storage provides different storage implementations for the game data.
// This package implements the Repository interface from the game package.
package storage

import (
	"context"
	"fmt"
	"sort"
	"sync"

	"coinflip-game/internal/game"
)

// MemoryRepository implements the Repository interface using in-memory storage.
// This is useful for testing and simple deployments where persistence is not required.
type MemoryRepository struct {
	mu      sync.RWMutex
	results map[string]*game.Result
	players map[string]*game.Player
}

// NewMemoryRepository creates a new in-memory repository
func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		results: make(map[string]*game.Result),
		players: make(map[string]*game.Player),
	}
}

// SaveResult saves a game result to memory
func (r *MemoryRepository) SaveResult(ctx context.Context, result *game.Result) error {
	if result == nil {
		return fmt.Errorf("result cannot be nil")
	}

	if result.ID == "" {
		return fmt.Errorf("result ID cannot be empty")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Create a deep copy to avoid external mutations
	resultCopy := &game.Result{
		ID:        result.ID,
		Side:      result.Side,
		Won:       result.Won,
		Payout:    result.Payout,
		Timestamp: result.Timestamp,
		Seed:      result.Seed,
	}

	// Deep copy the bet if it exists
	if result.Bet != nil {
		resultCopy.Bet = &game.Bet{
			ID:        result.Bet.ID,
			Amount:    result.Bet.Amount,
			Choice:    result.Bet.Choice,
			Timestamp: result.Bet.Timestamp,
		}
	}

	r.results[result.ID] = resultCopy
	return nil
}

// GetResults retrieves the most recent game results up to the specified limit
func (r *MemoryRepository) GetResults(ctx context.Context, limit int) ([]*game.Result, error) {
	if limit <= 0 {
		return []*game.Result{}, nil
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	// Convert map to slice for sorting
	results := make([]*game.Result, 0, len(r.results))
	for _, result := range r.results {
		// Create copies to avoid external mutations
		resultCopy := &game.Result{
			ID:        result.ID,
			Side:      result.Side,
			Won:       result.Won,
			Payout:    result.Payout,
			Timestamp: result.Timestamp,
			Seed:      result.Seed,
		}

		if result.Bet != nil {
			resultCopy.Bet = &game.Bet{
				ID:        result.Bet.ID,
				Amount:    result.Bet.Amount,
				Choice:    result.Bet.Choice,
				Timestamp: result.Bet.Timestamp,
			}
		}

		results = append(results, resultCopy)
	}

	// Sort by timestamp descending (most recent first)
	sort.Slice(results, func(i, j int) bool {
		return results[i].Timestamp.After(results[j].Timestamp)
	})

	// Apply limit
	if limit > len(results) {
		limit = len(results)
	}

	return results[:limit], nil
}

// GetStats calculates and returns statistics for a player based on their game history
func (r *MemoryRepository) GetStats(ctx context.Context, playerID string) (*game.Stats, error) {
	if playerID == "" {
		return nil, fmt.Errorf("player ID cannot be empty")
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	// Find the player to get current stats
	player, exists := r.players[playerID]
	if !exists {
		// Return empty stats for new players
		return &game.Stats{}, nil
	}

	// Return a copy of the stats to avoid external mutations
	statsCopy := game.Stats{
		GamesPlayed:   player.Stats.GamesPlayed,
		GamesWon:      player.Stats.GamesWon,
		TotalWagered:  player.Stats.TotalWagered,
		TotalWinnings: player.Stats.TotalWinnings,
		NetProfit:     player.Stats.NetProfit,
		WinRate:       player.Stats.WinRate,
	}

	return &statsCopy, nil
}

// SavePlayer saves or updates a player in memory
func (r *MemoryRepository) SavePlayer(ctx context.Context, player *game.Player) error {
	if player == nil {
		return fmt.Errorf("player cannot be nil")
	}

	if player.ID == "" {
		return fmt.Errorf("player ID cannot be empty")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Create a deep copy to avoid external mutations
	playerCopy := &game.Player{
		ID:      player.ID,
		Balance: player.Balance,
		Stats: game.Stats{
			GamesPlayed:   player.Stats.GamesPlayed,
			GamesWon:      player.Stats.GamesWon,
			TotalWagered:  player.Stats.TotalWagered,
			TotalWinnings: player.Stats.TotalWinnings,
			NetProfit:     player.Stats.NetProfit,
			WinRate:       player.Stats.WinRate,
		},
	}

	r.players[player.ID] = playerCopy
	return nil
}

// GetPlayer retrieves a player by ID from memory
func (r *MemoryRepository) GetPlayer(ctx context.Context, playerID string) (*game.Player, error) {
	if playerID == "" {
		return nil, fmt.Errorf("player ID cannot be empty")
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	player, exists := r.players[playerID]
	if !exists {
		return nil, fmt.Errorf("player not found: %s", playerID)
	}

	// Return a copy to avoid external mutations
	playerCopy := &game.Player{
		ID:      player.ID,
		Balance: player.Balance,
		Stats: game.Stats{
			GamesPlayed:   player.Stats.GamesPlayed,
			GamesWon:      player.Stats.GamesWon,
			TotalWagered:  player.Stats.TotalWagered,
			TotalWinnings: player.Stats.TotalWinnings,
			NetProfit:     player.Stats.NetProfit,
			WinRate:       player.Stats.WinRate,
		},
	}

	return playerCopy, nil
}

// Clear removes all data from the repository (useful for testing)
func (r *MemoryRepository) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.results = make(map[string]*game.Result)
	r.players = make(map[string]*game.Player)
}

// GetResultCount returns the total number of results stored
func (r *MemoryRepository) GetResultCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.results)
}

// GetPlayerCount returns the total number of players stored
func (r *MemoryRepository) GetPlayerCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.players)
}

// Package network provides room management for multiplayer coin flip games
package network

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"

	"coinflip-game/internal/game"
)

// Room constants
const (
	DefaultMinPlayers    = 2
	DefaultMaxPlayers    = 8
	BettingPhaseDuration = 60 * time.Second
	ResultPhaseDuration  = 10 * time.Second
	DefaultRoomTimeout   = 30 * time.Minute
)

// Common errors
var (
	ErrRoomFull        = errors.New("room is full")
	ErrRoomNotFound    = errors.New("room not found")
	ErrPlayerNotFound  = errors.New("player not found in room")
	ErrInvalidGamePhase = errors.New("invalid action for current game phase")
	ErrBettingClosed   = errors.New("betting phase has ended")
	ErrPlayerAlreadyBet = errors.New("player has already placed a bet this round")
)

// GameRoom represents a multiplayer game room
type GameRoom struct {
	mu            sync.RWMutex
	id            string
	name          string
	players       map[string]*RoomPlayer
	gameState     GameState
	currentRound  *GameRound
	config        *RoomConfig
	logger        *zap.Logger
	
	// Game timer
	timer         *time.Timer
	timerEnd      time.Time
	
	// Event channels
	eventChan     chan *Message
	stopChan      chan struct{}
	
	// Game statistics
	totalRounds   int
	createdAt     time.Time
	lastActivity  time.Time
}

// RoomPlayer represents a player in a room
type RoomPlayer struct {
	ID           string
	Name         string
	Balance      float64
	IsReady      bool
	IsOnline     bool
	LastSeen     time.Time
	CurrentBet   *BetData
	TotalGames   int
	TotalWins    int
	NetProfit    float64
}

// GameRound represents a single game round
type GameRound struct {
	ID           string
	StartTime    time.Time
	Bets         map[string]*BetData
	SeedCommits  map[string]string
	SeedReveals  map[string]string
	FinalSeed    string
	CoinResult   game.Side
	Results      map[string]*PlayerResult
	State        GameState
}

// RoomConfig contains room configuration
type RoomConfig struct {
	MinPlayers       int
	MaxPlayers       int
	MinBet           float64
	MaxBet           float64
	PayoutRatio      float64
	BettingDuration  time.Duration
	ResultDuration   time.Duration
	RequireConsensus bool
}

// DefaultRoomConfig returns default room configuration
func DefaultRoomConfig() *RoomConfig {
	return &RoomConfig{
		MinPlayers:       DefaultMinPlayers,
		MaxPlayers:       DefaultMaxPlayers,
		MinBet:           1.0,
		MaxBet:           100.0,
		PayoutRatio:      2.0,
		BettingDuration:  BettingPhaseDuration,
		ResultDuration:   ResultPhaseDuration,
		RequireConsensus: true,
	}
}

// NewGameRoom creates a new game room
func NewGameRoom(id, name string, config *RoomConfig, logger *zap.Logger) *GameRoom {
	if config == nil {
		config = DefaultRoomConfig()
	}
	
	room := &GameRoom{
		id:           id,
		name:         name,
		players:      make(map[string]*RoomPlayer),
		gameState:    StateWaiting,
		config:       config,
		logger:       logger,
		eventChan:    make(chan *Message, 100),
		stopChan:     make(chan struct{}),
		createdAt:    time.Now(),
		lastActivity: time.Now(),
	}
	
	return room
}

// ID returns the room ID
func (r *GameRoom) ID() string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.id
}

// Name returns the room name
func (r *GameRoom) Name() string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.name
}

// AddPlayer adds a player to the room
func (r *GameRoom) AddPlayer(playerID, playerName string, balance float64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	if len(r.players) >= r.config.MaxPlayers {
		return ErrRoomFull
	}
	
	player := &RoomPlayer{
		ID:       playerID,
		Name:     playerName,
		Balance:  balance,
		IsReady:  false,
		IsOnline: true,
		LastSeen: time.Now(),
	}
	
	r.players[playerID] = player
	r.lastActivity = time.Now()
	
	r.logger.Info("Player joined room",
		zap.String("room_id", r.id),
		zap.String("player_id", playerID),
		zap.String("player_name", playerName),
		zap.Int("total_players", len(r.players)),
	)
	
	// Send room update to all players
	r.broadcastRoomUpdate()
	
	// Auto-start betting if we have enough players and game is waiting
	r.checkAndStartGame()
	
	return nil
}

// RemovePlayer removes a player from the room
func (r *GameRoom) RemovePlayer(playerID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	player, exists := r.players[playerID]
	if !exists {
		return ErrPlayerNotFound
	}
	
	// Cancel any active bet
	if r.currentRound != nil && r.currentRound.Bets[playerID] != nil {
		// Refund the bet
		player.Balance += r.currentRound.Bets[playerID].Amount
		delete(r.currentRound.Bets, playerID)
	}
	
	delete(r.players, playerID)
	r.lastActivity = time.Now()
	
	r.logger.Info("Player left room",
		zap.String("room_id", r.id),
		zap.String("player_id", playerID),
		zap.Int("remaining_players", len(r.players)),
	)
	
	// Check if we need to pause the game
	if len(r.players) < r.config.MinPlayers && r.gameState == StateBetting {
		r.pauseGame()
	}
	
	r.broadcastRoomUpdate()
	return nil
}

// PlaceBet allows a player to place a bet
func (r *GameRoom) PlaceBet(playerID string, amount float64, choice game.Side) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	if r.gameState != StateBetting {
		return ErrInvalidGamePhase
	}
	
	player, exists := r.players[playerID]
	if !exists {
		return ErrPlayerNotFound
	}
	
	if r.currentRound == nil {
		return errors.New("no active round")
	}
	
	// Check if player already has a bet
	if r.currentRound.Bets[playerID] != nil {
		return ErrPlayerAlreadyBet
	}
	
	// Validate bet amount
	if amount < r.config.MinBet || amount > r.config.MaxBet {
		return game.ErrInvalidBetAmount
	}
	
	if player.Balance < amount {
		return game.ErrInsufficientBalance
	}
	
	// Create bet
	bet := &BetData{
		PlayerID: playerID,
		Amount:   amount,
		Choice:   choice,
		BetID:    r.generateBetID(),
	}
	
	// Deduct from balance and add bet
	player.Balance -= amount
	player.CurrentBet = bet
	r.currentRound.Bets[playerID] = bet
	r.lastActivity = time.Now()
	
	r.logger.Info("Bet placed",
		zap.String("room_id", r.id),
		zap.String("player_id", playerID),
		zap.Float64("amount", amount),
		zap.String("choice", choice.String()),
	)
	
	// Broadcast bet placement
	r.broadcastMessage(NewMessage(MsgBetPlaced, r.id, playerID, bet))
	
	// Broadcast updated room state with new player balances
	r.broadcastRoomUpdate()
	
	return nil
}

// StartGame starts a new game round
func (r *GameRoom) StartGame() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	if len(r.players) < r.config.MinPlayers {
		return errors.New("not enough players to start game")
	}
	
	if r.gameState != StateWaiting {
		return ErrInvalidGamePhase
	}
	
	// Create new round
	r.currentRound = &GameRound{
		ID:          r.generateRoundID(),
		StartTime:   time.Now(),
		Bets:        make(map[string]*BetData),
		SeedCommits: make(map[string]string),
		SeedReveals: make(map[string]string),
		Results:     make(map[string]*PlayerResult),
		State:       StateBetting,
	}
	
	r.gameState = StateBetting
	r.totalRounds++
	
	// Start betting timer
	r.startBettingPhase()
	
	r.logger.Info("Game round started",
		zap.String("room_id", r.id),
		zap.String("round_id", r.currentRound.ID),
		zap.Int("players", len(r.players)),
	)
	
	r.broadcastMessage(NewMessage(MsgGameStart, r.id, "", r.currentRound.ID))
	
	return nil
}

// checkAndStartGame checks if we should start a new betting round
func (r *GameRoom) checkAndStartGame() {
	// Only start if we have enough players and are in waiting state
	if len(r.players) >= r.config.MinPlayers && r.gameState == StateWaiting {
		r.logger.Info("Auto-starting betting round",
			zap.String("room_id", r.id),
			zap.Int("player_count", len(r.players)),
			zap.Int("min_players", r.config.MinPlayers),
		)
		
		// Use existing StartGame function which handles everything properly
		go func() {
			if err := r.StartGame(); err != nil {
				r.logger.Error("Failed to auto-start game", zap.Error(err))
			}
		}()
	}
}

// startBettingPhase starts the betting phase with timer
func (r *GameRoom) startBettingPhase() {
	r.timerEnd = time.Now().Add(r.config.BettingDuration)
	
	if r.timer != nil {
		r.timer.Stop()
	}
	
	r.timer = time.AfterFunc(r.config.BettingDuration, func() {
		r.endBettingPhase()
	})
	
	// Start timer broadcast routine
	go r.broadcastTimer()
	
	r.broadcastMessage(NewMessage(MsgBetPhase, r.id, "", TimerData{
		Phase:        StateBetting,
		SecondsLeft:  int(r.config.BettingDuration.Seconds()),
		TotalSeconds: int(r.config.BettingDuration.Seconds()),
	}))
}

// endBettingPhase ends the betting phase and starts revealing
func (r *GameRoom) endBettingPhase() {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	if r.gameState != StateBetting {
		return
	}
	
	r.gameState = StateRevealing
	
	r.logger.Info("Betting phase ended",
		zap.String("room_id", r.id),
		zap.String("round_id", r.currentRound.ID),
		zap.Int("total_bets", len(r.currentRound.Bets)),
	)
	
	// If no bets placed, return to waiting
	if len(r.currentRound.Bets) == 0 {
		r.gameState = StateWaiting
		r.currentRound = nil
		r.broadcastRoomUpdate()
		return
	}
	
	// Generate final seed and determine result
	r.generateFinalResult()
	
	// Start result phase
	r.startResultPhase()
}

// generateFinalResult generates the final coin flip result
func (r *GameRoom) generateFinalResult() {
	// Generate secure random seed
	seedBytes := make([]byte, 32)
	rand.Read(seedBytes)
	
	hash := sha256.Sum256(seedBytes)
	r.currentRound.FinalSeed = hex.EncodeToString(hash[:])
	
	// Determine coin result using the same logic as single-player
	rng := game.NewDefaultRandomGenerator()
	coinResult, _ := rng.FlipCoin(r.currentRound.FinalSeed)
	r.currentRound.CoinResult = coinResult
	
	// Calculate results for each bet
	for playerID, bet := range r.currentRound.Bets {
		player := r.players[playerID]
		won := bet.Choice == coinResult
		
		var payout float64
		if won {
			payout = bet.Amount * r.config.PayoutRatio
			player.Balance += payout
			player.TotalWins++
			player.NetProfit += (payout - bet.Amount)
		} else {
			player.NetProfit -= bet.Amount
		}
		
		player.TotalGames++
		player.CurrentBet = nil
		
		r.currentRound.Results[playerID] = &PlayerResult{
			PlayerID:   playerID,
			PlayerName: player.Name,
			Bet:        bet,
			Won:        won,
			Payout:     payout,
			NewBalance: player.Balance,
		}
	}
}

// startResultPhase starts the result display phase
func (r *GameRoom) startResultPhase() {
	r.gameState = StateResult
	
	// Prepare result data
	var winners, losers []PlayerResult
	for _, result := range r.currentRound.Results {
		if result.Won {
			winners = append(winners, *result)
		} else {
			losers = append(losers, *result)
		}
	}
	
	resultData := &GameResultData{
		RoundID:    r.currentRound.ID,
		CoinResult: r.currentRound.CoinResult,
		FinalSeed:  r.currentRound.FinalSeed,
		Winners:    winners,
		Losers:     losers,
		Timestamp:  time.Now(),
	}
	
	r.logger.Info("Game result generated",
		zap.String("room_id", r.id),
		zap.String("round_id", r.currentRound.ID),
		zap.String("coin_result", r.currentRound.CoinResult.String()),
		zap.Int("winners", len(winners)),
		zap.Int("losers", len(losers)),
	)
	
	// Broadcast result
	r.broadcastMessage(NewMessage(MsgGameResult, r.id, "", resultData))
	
	// Schedule return to waiting state
	time.AfterFunc(r.config.ResultDuration, func() {
		r.mu.Lock()
		defer r.mu.Unlock()
		
		r.gameState = StateWaiting
		r.currentRound = nil
		r.broadcastRoomUpdate()
		
		// Auto-start next round if enough players
		if len(r.players) >= r.config.MinPlayers {
			go func() {
				time.Sleep(2 * time.Second) // Brief pause between rounds
				r.StartGame()
			}()
		}
	})
}

// pauseGame pauses the current game
func (r *GameRoom) pauseGame() {
	if r.timer != nil {
		r.timer.Stop()
	}
	r.gameState = StatePaused
	
	r.logger.Info("Game paused", zap.String("room_id", r.id))
	r.broadcastRoomUpdate()
}

// broadcastTimer sends timer updates to all players
func (r *GameRoom) broadcastTimer() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			r.mu.RLock()
			if r.gameState != StateBetting {
				r.mu.RUnlock()
				return
			}
			
			secondsLeft := int(time.Until(r.timerEnd).Seconds())
			if secondsLeft <= 0 {
				r.mu.RUnlock()
				return
			}
			
			timerData := TimerData{
				Phase:        StateBetting,
				SecondsLeft:  secondsLeft,
				TotalSeconds: int(r.config.BettingDuration.Seconds()),
			}
			r.mu.RUnlock()
			
			r.broadcastMessage(NewMessage(MsgTimerUpdate, r.id, "", timerData))
			
		case <-r.stopChan:
			return
		}
	}
}

// broadcastRoomUpdate sends room state to all players
func (r *GameRoom) broadcastRoomUpdate() {
	players := make([]PlayerInfo, 0, len(r.players))
	for _, player := range r.players {
		players = append(players, PlayerInfo{
			ID:       player.ID,
			Name:     player.Name,
			Balance:  player.Balance,
			IsReady:  player.IsReady,
			HasBet:   player.CurrentBet != nil,
			IsOnline: player.IsOnline,
		})
	}
	
	updateData := &RoomUpdateData{
		RoomID:     r.id,
		Players:    players,
		GameState:  r.gameState,
		Timer:      int(time.Until(r.timerEnd).Seconds()),
		MinPlayers: r.config.MinPlayers,
		MaxPlayers: r.config.MaxPlayers,
	}
	
	r.broadcastMessage(NewMessage(MsgRoomUpdate, r.id, "", updateData))
}

// broadcastMessage sends a message to all players in the room
func (r *GameRoom) broadcastMessage(msg *Message) {
	select {
	case r.eventChan <- msg:
	default:
		r.logger.Warn("Event channel full, dropping message",
			zap.String("room_id", r.id),
			zap.String("message_type", string(msg.Type)),
		)
	}
}

// GetEventChannel returns the event channel for this room
func (r *GameRoom) GetEventChannel() <-chan *Message {
	return r.eventChan
}

// Stop stops the room and cleans up resources
func (r *GameRoom) Stop() {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	if r.timer != nil {
		r.timer.Stop()
	}
	
	close(r.stopChan)
	close(r.eventChan)
	
	r.logger.Info("Room stopped", zap.String("room_id", r.id))
}

// GetPlayers returns current players in the room
func (r *GameRoom) GetPlayers() map[string]*RoomPlayer {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	players := make(map[string]*RoomPlayer)
	for id, player := range r.players {
		players[id] = player
	}
	return players
}

// GetGameState returns the current game state
func (r *GameRoom) GetGameState() GameState {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.gameState
}

// Helper functions
func (r *GameRoom) generateBetID() string {
	return fmt.Sprintf("bet_%d", time.Now().UnixNano())
}

func (r *GameRoom) generateRoundID() string {
	return fmt.Sprintf("round_%s_%d", r.id, time.Now().UnixNano())
}
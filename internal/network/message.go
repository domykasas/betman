// Package network provides P2P networking functionality for multiplayer coin flip games.
// This package handles room management, player synchronization, and real-time communication.
package network

import (
	"encoding/json"
	"time"

	"coinflip-game/internal/game"
)

// MessageType represents different types of network messages
type MessageType string

const (
	// Room management messages
	MsgJoinRoom    MessageType = "join_room"
	MsgLeaveRoom   MessageType = "leave_room"
	MsgRoomUpdate  MessageType = "room_update"
	MsgPlayerList  MessageType = "player_list"
	
	// Game flow messages
	MsgGameStart   MessageType = "game_start"
	MsgBetPhase    MessageType = "bet_phase"
	MsgBetPlaced   MessageType = "bet_placed"
	MsgRevealPhase MessageType = "reveal_phase"
	MsgGameResult  MessageType = "game_result"
	MsgRoundEnd    MessageType = "round_end"
	
	// Synchronization messages
	MsgTimerUpdate MessageType = "timer_update"
	MsgSeedCommit  MessageType = "seed_commit"
	MsgSeedReveal  MessageType = "seed_reveal"
	
	// Error handling
	MsgError       MessageType = "error"
)

// Message represents a network message between peers
type Message struct {
	Type      MessageType `json:"type"`
	RoomID    string      `json:"room_id"`
	PlayerID  string      `json:"player_id"`
	Timestamp time.Time   `json:"timestamp"`
	Data      interface{} `json:"data"`
}

// RoomJoinData contains information for joining a room
type RoomJoinData struct {
	PlayerName string  `json:"player_name"`
	Balance    float64 `json:"balance"`
}

// RoomUpdateData contains current room state
type RoomUpdateData struct {
	RoomID      string       `json:"room_id"`
	Players     []PlayerInfo `json:"players"`
	GameState   GameState    `json:"game_state"`
	Timer       int          `json:"timer_seconds"`
	MinPlayers  int          `json:"min_players"`
	MaxPlayers  int          `json:"max_players"`
}

// PlayerInfo contains public player information
type PlayerInfo struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	Balance  float64 `json:"balance"`
	IsReady  bool    `json:"is_ready"`
	HasBet   bool    `json:"has_bet"`
	IsOnline bool    `json:"is_online"`
}

// GameState represents the current state of a multiplayer game
type GameState string

const (
	StateWaiting   GameState = "waiting"    // Waiting for players to join
	StateBetting   GameState = "betting"    // Players can place bets (60s timer)
	StateRevealing GameState = "revealing"  // Revealing coin flip result
	StateResult    GameState = "result"     // Showing results and payouts
	StatePaused    GameState = "paused"     // Game temporarily paused
)

// BetData contains betting information
type BetData struct {
	PlayerID string     `json:"player_id"`
	Amount   float64    `json:"amount"`
	Choice   game.Side  `json:"choice"`
	BetID    string     `json:"bet_id"`
}

// TimerData contains timer information
type TimerData struct {
	Phase         GameState `json:"phase"`
	SecondsLeft   int       `json:"seconds_left"`
	TotalSeconds  int       `json:"total_seconds"`
}

// SeedCommitData contains committed seed hash for consensus
type SeedCommitData struct {
	PlayerID   string `json:"player_id"`
	SeedHash   string `json:"seed_hash"`
	RoundID    string `json:"round_id"`
}

// SeedRevealData contains revealed seed for verification
type SeedRevealData struct {
	PlayerID string `json:"player_id"`
	Seed     string `json:"seed"`
	RoundID  string `json:"round_id"`
}

// GameResultData contains the final game result
type GameResultData struct {
	RoundID    string           `json:"round_id"`
	CoinResult game.Side        `json:"coin_result"`
	FinalSeed  string           `json:"final_seed"`
	Winners    []PlayerResult   `json:"winners"`
	Losers     []PlayerResult   `json:"losers"`
	Timestamp  time.Time        `json:"timestamp"`
}

// PlayerResult contains individual player's result
type PlayerResult struct {
	PlayerID     string     `json:"player_id"`
	PlayerName   string     `json:"player_name"`
	Bet          *BetData   `json:"bet,omitempty"`
	Won          bool       `json:"won"`
	Payout       float64    `json:"payout"`
	NewBalance   float64    `json:"new_balance"`
}

// ErrorData contains error information
type ErrorData struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// NewMessage creates a new network message
func NewMessage(msgType MessageType, roomID, playerID string, data interface{}) *Message {
	return &Message{
		Type:      msgType,
		RoomID:    roomID,
		PlayerID:  playerID,
		Timestamp: time.Now(),
		Data:      data,
	}
}

// ToJSON serializes the message to JSON
func (m *Message) ToJSON() ([]byte, error) {
	return json.Marshal(m)
}

// FromJSON deserializes a message from JSON
func FromJSON(data []byte) (*Message, error) {
	var msg Message
	err := json.Unmarshal(data, &msg)
	return &msg, err
}

// GetData attempts to unmarshal the Data field into the provided type
func (m *Message) GetData(target interface{}) error {
	dataBytes, err := json.Marshal(m.Data)
	if err != nil {
		return err
	}
	return json.Unmarshal(dataBytes, target)
}
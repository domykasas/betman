// Package network provides WebSocket client functionality for multiplayer games
package network

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"

	"coinflip-game/internal/game"
)

// NetworkClient handles WebSocket connection to the multiplayer server
type NetworkClient struct {
	mu           sync.RWMutex
	conn         *websocket.Conn
	serverURL    string
	playerID     string
	playerName   string
	currentRoom  string
	logger       *zap.Logger
	
	// Event handling
	messageHandlers map[MessageType]func(*Message)
	eventChan       chan *Message
	errorChan       chan error
	
	// Connection state
	connected       bool
	reconnectDelay  time.Duration
	maxReconnects   int
	reconnectCount  int
	
	// Context for graceful shutdown
	ctx             context.Context
	cancel          context.CancelFunc
	
	// Ping/pong for connection health
	pingPeriod      time.Duration
	pongWait        time.Duration
	writeWait       time.Duration
}

// ClientConfig contains client configuration
type ClientConfig struct {
	ServerURL       string
	ReconnectDelay  time.Duration
	MaxReconnects   int
	PingPeriod      time.Duration
	PongWait        time.Duration
	WriteWait       time.Duration
	ReadBufferSize  int
	WriteBufferSize int
}

// DefaultClientConfig returns default client configuration
func DefaultClientConfig() *ClientConfig {
	return &ClientConfig{
		ServerURL:       "ws://localhost:8080/ws",
		ReconnectDelay:  5 * time.Second,
		MaxReconnects:   5,
		PingPeriod:      54 * time.Second,
		PongWait:        60 * time.Second,
		WriteWait:       10 * time.Second,
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}
}

// NewNetworkClient creates a new network client
func NewNetworkClient(config *ClientConfig, playerID, playerName string, logger *zap.Logger) *NetworkClient {
	if config == nil {
		config = DefaultClientConfig()
	}
	
	ctx, cancel := context.WithCancel(context.Background())
	
	client := &NetworkClient{
		serverURL:       config.ServerURL,
		playerID:        playerID,
		playerName:      playerName,
		logger:          logger,
		messageHandlers: make(map[MessageType]func(*Message)),
		eventChan:       make(chan *Message, 100),
		errorChan:       make(chan error, 10),
		reconnectDelay:  config.ReconnectDelay,
		maxReconnects:   config.MaxReconnects,
		pingPeriod:      config.PingPeriod,
		pongWait:        config.PongWait,
		writeWait:       config.WriteWait,
		ctx:             ctx,
		cancel:          cancel,
	}
	
	// Set up default message handlers
	client.setupDefaultHandlers()
	
	return client
}

// Connect establishes connection to the server
func (c *NetworkClient) Connect() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	if c.connected {
		return nil
	}
	
	u, err := url.Parse(c.serverURL)
	if err != nil {
		return fmt.Errorf("invalid server URL: %w", err)
	}
	
	c.logger.Info("Connecting to server", zap.String("url", c.serverURL))
	
	dialer := websocket.DefaultDialer
	conn, _, err := dialer.Dial(u.String(), nil)
	if err != nil {
		return fmt.Errorf("failed to connect to server: %w", err)
	}
	
	c.conn = conn
	c.connected = true
	c.reconnectCount = 0
	
	// Set connection options - increased for game result messages
	c.conn.SetReadLimit(4096)
	c.conn.SetReadDeadline(time.Now().Add(c.pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(c.pongWait))
		return nil
	})
	
	// Start connection management goroutines
	go c.readPump()
	go c.writePump()
	go c.pingPump()
	
	c.logger.Info("Connected to server successfully")
	return nil
}

// Disconnect closes the connection to the server
func (c *NetworkClient) Disconnect() {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	if !c.connected {
		return
	}
	
	c.cancel()
	c.connected = false
	
	if c.conn != nil {
		c.conn.Close()
		c.conn = nil
	}
	
	c.logger.Info("Disconnected from server")
}

// JoinRoom joins a multiplayer room
func (c *NetworkClient) JoinRoom(roomID string, balance float64) error {
	if !c.IsConnected() {
		return errors.New("not connected to server")
	}
	
	joinData := RoomJoinData{
		PlayerName: c.playerName,
		Balance:    balance,
	}
	
	msg := NewMessage(MsgJoinRoom, roomID, c.playerID, joinData)
	
	if err := c.sendMessage(msg); err != nil {
		return fmt.Errorf("failed to send join room message: %w", err)
	}
	
	c.mu.Lock()
	c.currentRoom = roomID
	c.mu.Unlock()
	
	c.logger.Info("Joining room", 
		zap.String("room_id", roomID),
		zap.String("player_name", c.playerName),
	)
	
	return nil
}

// LeaveRoom leaves the current room
func (c *NetworkClient) LeaveRoom() error {
	c.mu.RLock()
	roomID := c.currentRoom
	c.mu.RUnlock()
	
	if roomID == "" {
		return nil
	}
	
	if !c.IsConnected() {
		return errors.New("not connected to server")
	}
	
	msg := NewMessage(MsgLeaveRoom, roomID, c.playerID, nil)
	
	if err := c.sendMessage(msg); err != nil {
		return fmt.Errorf("failed to send leave room message: %w", err)
	}
	
	c.mu.Lock()
	c.currentRoom = ""
	c.mu.Unlock()
	
	c.logger.Info("Left room", zap.String("room_id", roomID))
	return nil
}

// PlaceBet places a bet in the current room
func (c *NetworkClient) PlaceBet(amount float64, choice game.Side) error {
	c.mu.RLock()
	roomID := c.currentRoom
	c.mu.RUnlock()
	
	if roomID == "" {
		return errors.New("not in a room")
	}
	
	if !c.IsConnected() {
		return errors.New("not connected to server")
	}
	
	betData := BetData{
		PlayerID: c.playerID,
		Amount:   amount,
		Choice:   choice,
		BetID:    fmt.Sprintf("bet_%d", time.Now().UnixNano()),
	}
	
	msg := NewMessage(MsgBetPlaced, roomID, c.playerID, betData)
	
	if err := c.sendMessage(msg); err != nil {
		return fmt.Errorf("failed to send bet message: %w", err)
	}
	
	c.logger.Info("Placed bet",
		zap.String("room_id", roomID),
		zap.Float64("amount", amount),
		zap.String("choice", choice.String()),
	)
	
	return nil
}

// IsConnected returns whether the client is connected
func (c *NetworkClient) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.connected
}

// GetCurrentRoom returns the current room ID
func (c *NetworkClient) GetCurrentRoom() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.currentRoom
}

// SetMessageHandler sets a handler for a specific message type
func (c *NetworkClient) SetMessageHandler(msgType MessageType, handler func(*Message)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.messageHandlers[msgType] = handler
}

// GetEventChannel returns the event channel
func (c *NetworkClient) GetEventChannel() <-chan *Message {
	return c.eventChan
}

// GetErrorChannel returns the error channel
func (c *NetworkClient) GetErrorChannel() <-chan error {
	return c.errorChan
}

// setupDefaultHandlers sets up default message handlers
func (c *NetworkClient) setupDefaultHandlers() {
	c.messageHandlers[MsgError] = func(msg *Message) {
		var errorData ErrorData
		if err := msg.GetData(&errorData); err == nil {
			c.logger.Error("Server error",
				zap.String("code", errorData.Code),
				zap.String("message", errorData.Message),
			)
		}
	}
	
	c.messageHandlers[MsgRoomUpdate] = func(msg *Message) {
		c.logger.Debug("Room update received", zap.String("room_id", msg.RoomID))
	}
	
	c.messageHandlers[MsgGameResult] = func(msg *Message) {
		c.logger.Info("Game result received", zap.String("room_id", msg.RoomID))
	}
}

// sendMessage sends a message to the server
func (c *NetworkClient) sendMessage(msg *Message) error {
	if !c.connected || c.conn == nil {
		return errors.New("not connected")
	}
	
	data, err := msg.ToJSON()
	if err != nil {
		return fmt.Errorf("failed to serialize message: %w", err)
	}
	
	c.conn.SetWriteDeadline(time.Now().Add(c.writeWait))
	return c.conn.WriteMessage(websocket.TextMessage, data)
}

// readPump handles reading messages from the WebSocket
func (c *NetworkClient) readPump() {
	defer func() {
		c.handleDisconnect()
	}()
	
	for {
		select {
		case <-c.ctx.Done():
			return
		default:
			_, messageBytes, err := c.conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					c.logger.Error("WebSocket read error", zap.Error(err))
				}
				return
			}
			
			c.handleMessage(messageBytes)
		}
	}
}

// writePump handles writing messages to the WebSocket
func (c *NetworkClient) writePump() {
	pingPeriod := c.pingPeriod
	if pingPeriod <= 0 {
		pingPeriod = 54 * time.Second // Default fallback
	}
	ticker := time.NewTicker(pingPeriod)
	defer ticker.Stop()
	
	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			// Ping is handled by pingPump
		}
	}
}

// pingPump sends periodic ping messages
func (c *NetworkClient) pingPump() {
	pingPeriod := c.pingPeriod
	if pingPeriod <= 0 {
		pingPeriod = 54 * time.Second // Default fallback
	}
	ticker := time.NewTicker(pingPeriod)
	defer ticker.Stop()
	
	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			c.mu.RLock()
			conn := c.conn
			connected := c.connected
			c.mu.RUnlock()
			
			if !connected || conn == nil {
				return
			}
			
			conn.SetWriteDeadline(time.Now().Add(c.writeWait))
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				c.logger.Error("Failed to send ping", zap.Error(err))
				return
			}
		}
	}
}

// handleMessage processes incoming messages
func (c *NetworkClient) handleMessage(messageBytes []byte) {
	var msg Message
	if err := json.Unmarshal(messageBytes, &msg); err != nil {
		c.logger.Error("Failed to parse message", zap.Error(err))
		return
	}
	
	// Send to event channel
	select {
	case c.eventChan <- &msg:
	default:
		c.logger.Warn("Event channel full, dropping message")
	}
	
	// Call specific handler if available
	c.mu.RLock()
	if handler, exists := c.messageHandlers[msg.Type]; exists {
		c.mu.RUnlock()
		handler(&msg)
	} else {
		c.mu.RUnlock()
		c.logger.Debug("No handler for message type", zap.String("type", string(msg.Type)))
	}
}

// handleDisconnect handles connection loss and potential reconnection
func (c *NetworkClient) handleDisconnect() {
	c.mu.Lock()
	c.connected = false
	if c.conn != nil {
		c.conn.Close()
		c.conn = nil
	}
	c.mu.Unlock()
	
	c.logger.Warn("Connection lost")
	
	// Send error to error channel
	select {
	case c.errorChan <- errors.New("connection lost"):
	default:
	}
	
	// Attempt reconnection if configured
	if c.maxReconnects > 0 && c.reconnectCount < c.maxReconnects {
		go c.attemptReconnect()
	}
}

// attemptReconnect attempts to reconnect to the server
func (c *NetworkClient) attemptReconnect() {
	c.reconnectCount++
	
	c.logger.Info("Attempting to reconnect",
		zap.Int("attempt", c.reconnectCount),
		zap.Int("max_attempts", c.maxReconnects),
	)
	
	time.Sleep(c.reconnectDelay)
	
	if err := c.Connect(); err != nil {
		c.logger.Error("Reconnection failed", zap.Error(err))
		
		if c.reconnectCount < c.maxReconnects {
			go c.attemptReconnect()
		} else {
			select {
			case c.errorChan <- errors.New("max reconnection attempts reached"):
			default:
			}
		}
		return
	}
	
	// Re-join room if we were in one
	c.mu.RLock()
	roomID := c.currentRoom
	c.mu.RUnlock()
	
	if roomID != "" {
		if err := c.JoinRoom(roomID, 1000); err != nil {
			c.logger.Error("Failed to rejoin room after reconnect", zap.Error(err))
		}
	}
}
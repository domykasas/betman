// Package network provides WebSocket server functionality for multiplayer games
package network

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

// Server manages WebSocket connections and game rooms
type Server struct {
	mu        sync.RWMutex
	rooms     map[string]*GameRoom
	clients   map[*Client]*GameRoom
	upgrader  websocket.Upgrader
	logger    *zap.Logger
	
	// Server configuration
	config    *ServerConfig
	
	// Channels
	register   chan *Client
	unregister chan *Client
	broadcast  chan []byte
	
	// Context for graceful shutdown
	ctx        context.Context
	cancel     context.CancelFunc
}

// Client represents a WebSocket client connection
type Client struct {
	conn     *websocket.Conn
	server   *Server
	room     *GameRoom
	playerID string
	name     string
	send     chan []byte
	mu       sync.RWMutex
}

// ServerConfig contains server configuration
type ServerConfig struct {
	Host            string
	Port            int
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	MaxMessageSize  int64
	PingPeriod      time.Duration
	PongWait        time.Duration
	MaxRooms        int
	MaxClientsRoom  int
	CleanupInterval time.Duration
}

// DefaultServerConfig returns default server configuration
func DefaultServerConfig() *ServerConfig {
	return &ServerConfig{
		Host:            "localhost",
		Port:            8080,
		ReadTimeout:     60 * time.Second,
		WriteTimeout:    10 * time.Second,
		MaxMessageSize:  4096, // Increased for game result messages
		PingPeriod:      54 * time.Second,
		PongWait:        60 * time.Second,
		MaxRooms:        100,
		MaxClientsRoom:  8,
		CleanupInterval: 5 * time.Minute,
	}
}

// NewServer creates a new WebSocket server
func NewServer(config *ServerConfig, logger *zap.Logger) *Server {
	if config == nil {
		config = DefaultServerConfig()
	}
	
	ctx, cancel := context.WithCancel(context.Background())
	
	server := &Server{
		rooms:      make(map[string]*GameRoom),
		clients:    make(map[*Client]*GameRoom),
		logger:     logger,
		config:     config,
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan []byte),
		ctx:        ctx,
		cancel:     cancel,
	}
	
	server.upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			// Allow all origins for development
			// In production, implement proper origin checking
			return true
		},
	}
	
	return server
}

// Start starts the WebSocket server
func (s *Server) Start() error {
	// Start the main event loop
	go s.run()
	
	// Start cleanup routine
	go s.cleanup()
	
	// Setup HTTP handlers
	http.HandleFunc("/ws", s.handleWebSocket)
	http.HandleFunc("/rooms", s.handleRooms)
	http.HandleFunc("/health", s.handleHealth)
	
	address := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)
	s.logger.Info("Starting WebSocket server", zap.String("address", address))
	
	return http.ListenAndServe(address, nil)
}

// Stop stops the server gracefully
func (s *Server) Stop() {
	s.cancel()
	
	s.mu.Lock()
	defer s.mu.Unlock()
	
	// Close all rooms
	for _, room := range s.rooms {
		room.Stop()
	}
	
	// Close all client connections
	for client := range s.clients {
		client.close()
	}
	
	s.logger.Info("Server stopped")
}

// run handles the main server event loop
func (s *Server) run() {
	pingPeriod := s.config.PingPeriod
	if pingPeriod <= 0 {
		pingPeriod = 54 * time.Second
	}
	ticker := time.NewTicker(pingPeriod)
	defer ticker.Stop()
	
	for {
		select {
		case client := <-s.register:
			s.registerClient(client)
			
		case client := <-s.unregister:
			s.unregisterClient(client)
			
		case message := <-s.broadcast:
			s.broadcastMessage(message)
			
		case <-ticker.C:
			s.pingClients()
			
		case <-s.ctx.Done():
			return
		}
	}
}

// handleWebSocket handles WebSocket connection upgrades
func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		s.logger.Error("Failed to upgrade connection", zap.Error(err))
		return
	}
	
	client := &Client{
		conn:   conn,
		server: s,
		send:   make(chan []byte, 256),
	}
	
	client.conn.SetReadLimit(s.config.MaxMessageSize)
	client.conn.SetReadDeadline(time.Now().Add(s.config.PongWait))
	client.conn.SetPongHandler(func(string) error {
		client.conn.SetReadDeadline(time.Now().Add(s.config.PongWait))
		return nil
	})
	
	s.register <- client
	
	// Start client goroutines
	go client.writePump()
	go client.readPump()
}

// handleRooms returns available rooms
func (s *Server) handleRooms(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	type RoomInfo struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		Players     int    `json:"players"`
		MaxPlayers  int    `json:"max_players"`
		GameState   string `json:"game_state"`
	}
	
	rooms := make([]RoomInfo, 0, len(s.rooms))
	for _, room := range s.rooms {
		players := room.GetPlayers()
		rooms = append(rooms, RoomInfo{
			ID:         room.ID(),
			Name:       room.Name(),
			Players:    len(players),
			MaxPlayers: room.config.MaxPlayers,
			GameState:  string(room.GetGameState()),
		})
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"rooms": rooms,
		"total": len(rooms),
	})
}

// handleHealth returns server health status
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":        "healthy",
		"active_rooms":  len(s.rooms),
		"active_clients": len(s.clients),
		"uptime":        time.Since(time.Now()).String(),
	})
}

// registerClient registers a new client
func (s *Server) registerClient(client *Client) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	s.clients[client] = nil
	s.logger.Info("Client connected", zap.String("remote_addr", client.conn.RemoteAddr().String()))
}

// unregisterClient unregisters a client
func (s *Server) unregisterClient(client *Client) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if room, exists := s.clients[client]; exists {
		delete(s.clients, client)
		
		// Remove from room if in one
		if room != nil && client.playerID != "" {
			room.RemovePlayer(client.playerID)
		}
		
		close(client.send)
		client.conn.Close()
		
		s.logger.Info("Client disconnected", 
			zap.String("player_id", client.playerID),
			zap.String("room_id", func() string {
				if room != nil {
					return room.ID()
				}
				return ""
			}()),
		)
	}
}

// broadcastMessage sends a message to all clients
func (s *Server) broadcastMessage(message []byte) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	for client := range s.clients {
		select {
		case client.send <- message:
		default:
			close(client.send)
			delete(s.clients, client)
		}
	}
}

// pingClients sends ping messages to all clients
func (s *Server) pingClients() {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	for client := range s.clients {
		select {
		case client.send <- []byte{}:
		default:
			close(client.send)
			delete(s.clients, client)
		}
	}
}

// cleanup removes empty rooms and inactive clients
func (s *Server) cleanup() {
	cleanupInterval := s.config.CleanupInterval
	if cleanupInterval <= 0 {
		cleanupInterval = 5 * time.Minute
	}
	ticker := time.NewTicker(cleanupInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			s.performCleanup()
		case <-s.ctx.Done():
			return
		}
	}
}

// performCleanup removes empty rooms
func (s *Server) performCleanup() {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	for roomID, room := range s.rooms {
		players := room.GetPlayers()
		if len(players) == 0 {
			room.Stop()
			delete(s.rooms, roomID)
			s.logger.Info("Removed empty room", zap.String("room_id", roomID))
		}
	}
}

// CreateRoom creates a new game room
func (s *Server) CreateRoom(roomID, roomName string, config *RoomConfig) (*GameRoom, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if len(s.rooms) >= s.config.MaxRooms {
		return nil, errors.New("maximum number of rooms reached")
	}
	
	if _, exists := s.rooms[roomID]; exists {
		return nil, errors.New("room already exists")
	}
	
	room := NewGameRoom(roomID, roomName, config, s.logger)
	s.rooms[roomID] = room
	
	// Start room event handling
	go s.handleRoomEvents(room)
	
	s.logger.Info("Room created", 
		zap.String("room_id", roomID),
		zap.String("room_name", roomName),
	)
	
	return room, nil
}

// GetRoom returns a room by ID
func (s *Server) GetRoom(roomID string) (*GameRoom, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	room, exists := s.rooms[roomID]
	return room, exists
}

// handleRoomEvents handles events from a game room
func (s *Server) handleRoomEvents(room *GameRoom) {
	for message := range room.GetEventChannel() {
		// Broadcast room events to all clients in the room
		s.broadcastToRoom(room, message)
	}
}

// broadcastToRoom sends a message to all clients in a specific room
func (s *Server) broadcastToRoom(room *GameRoom, message *Message) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	data, err := message.ToJSON()
	if err != nil {
		s.logger.Error("Failed to serialize message", zap.Error(err))
		return
	}
	
	for client, clientRoom := range s.clients {
		if clientRoom == room {
			select {
			case client.send <- data:
			default:
				close(client.send)
				delete(s.clients, client)
			}
		}
	}
}

// Client methods

// readPump handles reading messages from the WebSocket connection
func (c *Client) readPump() {
	defer func() {
		c.server.unregister <- c
		c.conn.Close()
	}()
	
	for {
		_, messageBytes, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				c.server.logger.Error("WebSocket error", zap.Error(err))
			}
			break
		}
		
		// Parse and handle the message
		c.handleMessage(messageBytes)
	}
}

// writePump handles writing messages to the WebSocket connection
func (c *Client) writePump() {
	ticker := time.NewTicker(c.server.config.PingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()
	
	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(c.server.config.WriteTimeout))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			
			if len(message) == 0 {
				// Ping message
				if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
					return
				}
			} else {
				// Regular message
				if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
					return
				}
			}
			
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(c.server.config.WriteTimeout))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// handleMessage processes incoming messages from clients
func (c *Client) handleMessage(messageBytes []byte) {
	var msg Message
	if err := json.Unmarshal(messageBytes, &msg); err != nil {
		c.server.logger.Error("Failed to parse message", zap.Error(err))
		c.sendError("invalid_message", "Failed to parse message")
		return
	}
	
	switch msg.Type {
	case MsgJoinRoom:
		c.handleJoinRoom(&msg)
	case MsgLeaveRoom:
		c.handleLeaveRoom(&msg)
	case MsgBetPlaced:
		c.handlePlaceBet(&msg)
	default:
		c.server.logger.Warn("Unknown message type", zap.String("type", string(msg.Type)))
	}
}

// handleJoinRoom handles room join requests
func (c *Client) handleJoinRoom(msg *Message) {
	var joinData RoomJoinData
	if err := msg.GetData(&joinData); err != nil {
		c.sendError("invalid_data", "Invalid join room data")
		return
	}
	
	// Get or create room
	room, exists := c.server.GetRoom(msg.RoomID)
	if !exists {
		// Auto-create room for development
		var err error
		room, err = c.server.CreateRoom(msg.RoomID, fmt.Sprintf("Room %s", msg.RoomID), DefaultRoomConfig())
		if err != nil {
			c.sendError("room_creation_failed", err.Error())
			return
		}
	}
	
	// Add player to room
	c.playerID = msg.PlayerID
	c.name = joinData.PlayerName
	if err := room.AddPlayer(msg.PlayerID, joinData.PlayerName, joinData.Balance); err != nil {
		c.sendError("join_failed", err.Error())
		return
	}
	
	// Update client-room mapping
	c.server.mu.Lock()
	c.server.clients[c] = room
	c.room = room
	c.server.mu.Unlock()
	
	c.server.logger.Info("Player joined room",
		zap.String("player_id", msg.PlayerID),
		zap.String("room_id", msg.RoomID),
	)
}

// handleLeaveRoom handles room leave requests
func (c *Client) handleLeaveRoom(msg *Message) {
	if c.room == nil {
		c.sendError("not_in_room", "Not currently in a room")
		return
	}
	
	c.room.RemovePlayer(c.playerID)
	
	c.server.mu.Lock()
	c.server.clients[c] = nil
	c.room = nil
	c.server.mu.Unlock()
}

// handlePlaceBet handles bet placement requests
func (c *Client) handlePlaceBet(msg *Message) {
	if c.room == nil {
		c.sendError("not_in_room", "Not currently in a room")
		return
	}
	
	var betData BetData
	if err := msg.GetData(&betData); err != nil {
		c.sendError("invalid_bet_data", "Invalid bet data")
		return
	}
	
	if err := c.room.PlaceBet(c.playerID, betData.Amount, betData.Choice); err != nil {
		c.sendError("bet_failed", err.Error())
		return
	}
}

// sendError sends an error message to the client
func (c *Client) sendError(code, message string) {
	errorMsg := NewMessage(MsgError, "", c.playerID, ErrorData{
		Code:    code,
		Message: message,
	})
	
	if data, err := errorMsg.ToJSON(); err == nil {
		select {
		case c.send <- data:
		default:
			// Channel full, client will be disconnected
		}
	}
}

// close closes the client connection
func (c *Client) close() {
	c.conn.Close()
}
// Package ui provides multiplayer GUI components for the coin flip game
package ui

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"go.uber.org/zap"

	"coinflip-game/internal/config"
	"coinflip-game/internal/game"
	"coinflip-game/internal/network"
)

// UIUpdate represents a UI update to be executed on the main thread
type UIUpdate struct {
	updateFunc func()
}

// PlayerStats tracks comprehensive player statistics
type PlayerStats struct {
	PlayerName    string
	TotalGames    int
	GamesWon      int
	GamesLost     int
	TotalBet      float64
	TotalWon      float64
	NetProfit     float64
	CurrentBalance float64
	LastSeen      time.Time
}

// MultiplayerGameUI manages the multiplayer game interface
type MultiplayerGameUI struct {
	ctx          context.Context
	app          fyne.App
	window       fyne.Window
	config       *config.Config
	logger       *zap.Logger
	networkClient *network.NetworkClient
	
	// Player info
	playerID     string
	playerName   string
	balance      float64
	
	// UI components
	connectionStatus *widget.Label
	roomInfo         *widget.Label
	playersList      *widget.List
	timerLabel       *widget.Label
	progressBar      *widget.ProgressBar
	
	betAmountEntry   *widget.Entry
	headsButton      *widget.Button
	tailsButton      *widget.Button
	
	gameResult       *widget.Label
	chatMessages     *widget.List
	chatEntry        *widget.Entry
	
	// History/Scoreboard components
	historyList      *widget.List
	scoreboardList   *widget.List
	
	// Room state
	currentPlayers   []network.PlayerInfo
	gameState        network.GameState
	timerSeconds     int
	totalSeconds     int
	
	// Game history and player statistics
	gameHistory      []*network.GameResultData
	playerStats      map[string]*PlayerStats
	
	// UI update channel for thread-safe updates
	uiUpdateChan     chan UIUpdate
}

// NewMultiplayerGameUI creates a new multiplayer game UI
func NewMultiplayerGameUI(ctx context.Context, app fyne.App, cfg *config.Config, logger *zap.Logger) *MultiplayerGameUI {
	// Generate unique player ID and name with suffix
	playerIDNano := time.Now().UnixNano()
	ui := &MultiplayerGameUI{
		ctx:          ctx,
		app:          app,
		config:       cfg,
		logger:       logger,
		playerID:     fmt.Sprintf("player_%d", playerIDNano),
		playerName:   fmt.Sprintf("Player%d", playerIDNano%10000), // Last 4 digits for readability
		balance:      cfg.Game.StartingBalance,
		gameHistory:  make([]*network.GameResultData, 0),
		playerStats:  make(map[string]*PlayerStats),
		uiUpdateChan: make(chan UIUpdate, 100), // Buffered channel for UI updates
	}
	
	ui.window = app.NewWindow("🎮 Multiplayer Coin Flip")
	ui.setupNetworking()
	ui.setupUI()
	
	// Start UI update processor on main thread
	go ui.processUIUpdates()
	
	return ui
}

// GetWindow returns the main application window
func (ui *MultiplayerGameUI) GetWindow() fyne.Window {
	return ui.window
}

// processUIUpdates processes UI updates on the main thread
func (ui *MultiplayerGameUI) processUIUpdates() {
	for {
		select {
		case <-ui.ctx.Done():
			return
		case update := <-ui.uiUpdateChan:
			// Ensure UI updates happen on Fyne's main thread
			fyne.Do(update.updateFunc)
		}
	}
}

// queueUIUpdate queues a UI update to be executed on the main thread
func (ui *MultiplayerGameUI) queueUIUpdate(updateFunc func()) {
	select {
	case ui.uiUpdateChan <- UIUpdate{updateFunc: updateFunc}:
		// Update queued successfully
	default:
		// Channel full, log warning but don't block
		ui.logger.Warn("UI update channel full, dropping update")
	}
}

// setupNetworking initializes the network client
func (ui *MultiplayerGameUI) setupNetworking() {
	// Start with default configuration to avoid zero values
	clientConfig := network.DefaultClientConfig()
	// Override only the server URL
	clientConfig.ServerURL = fmt.Sprintf("ws://%s:%d/ws", 
		ui.config.Multiplayer.ServerHost, 
		ui.config.Multiplayer.ServerPort)
	
	ui.networkClient = network.NewNetworkClient(clientConfig, ui.playerID, ui.playerName, ui.logger)
	
	// Set up message handlers
	ui.setupMessageHandlers()
	
	// Start event processing
	go ui.processNetworkEvents()
}

// setupMessageHandlers sets up handlers for network messages
func (ui *MultiplayerGameUI) setupMessageHandlers() {
	ui.networkClient.SetMessageHandler(network.MsgRoomUpdate, ui.handleRoomUpdate)
	ui.networkClient.SetMessageHandler(network.MsgTimerUpdate, ui.handleTimerUpdate)
	ui.networkClient.SetMessageHandler(network.MsgGameResult, ui.handleGameResult)
	ui.networkClient.SetMessageHandler(network.MsgBetPhase, ui.handleBetPhase)
	ui.networkClient.SetMessageHandler(network.MsgError, ui.handleError)
}

// processNetworkEvents processes network events
func (ui *MultiplayerGameUI) processNetworkEvents() {
	for {
		select {
		case <-ui.ctx.Done():
			return
		case err := <-ui.networkClient.GetErrorChannel():
			ui.logger.Error("Network error", zap.Error(err))
			// Queue UI update to be executed on main thread
			ui.queueUIUpdate(func() {
				ui.connectionStatus.SetText("❌ Disconnected: " + err.Error())
			})
		case <-ui.networkClient.GetEventChannel():
			// Events are handled by specific handlers
		}
	}
}

// setupUI creates and arranges all UI components
func (ui *MultiplayerGameUI) setupUI() {
	// Minimal connection status (no manual buttons - auto-connects)
	ui.connectionStatus = widget.NewLabel("🔄 Connecting...")
	ui.roomInfo = widget.NewLabel("Not in room")
	
	statusSection := container.NewVBox(
		ui.connectionStatus,
		ui.roomInfo,
	)
	
	// Prominent timer section - larger and more visible
	ui.timerLabel = widget.NewLabel("⏱️ Waiting for players...")
	ui.timerLabel.Alignment = fyne.TextAlignCenter
	ui.timerLabel.TextStyle = fyne.TextStyle{Bold: true}
	ui.progressBar = widget.NewProgressBar()
	ui.progressBar.SetValue(0)
	
	timerSection := container.NewVBox(
		widget.NewLabel("🕐 Game Timer"),
		ui.timerLabel,
		ui.progressBar,
		widget.NewSeparator(),
	)
	
	// Players list
	ui.playersList = widget.NewList(
		func() int { return len(ui.currentPlayers) },
		func() fyne.CanvasObject {
			return container.NewHBox(
				widget.NewLabel("Player"),
				widget.NewLabel("Status"),
				widget.NewLabel("Balance"),
			)
		},
		func(id widget.ListItemID, item fyne.CanvasObject) {
			if id >= len(ui.currentPlayers) {
				return
			}
			player := ui.currentPlayers[id]
			cont := item.(*fyne.Container)
			
			nameLabel := cont.Objects[0].(*widget.Label)
			statusLabel := cont.Objects[1].(*widget.Label)
			balanceLabel := cont.Objects[2].(*widget.Label)
			
			nameLabel.SetText(player.Name)
			
			status := "⚪"
			if player.IsOnline {
				status = "🟢"
			}
			if player.HasBet {
				status += " 🎲"
			}
			statusLabel.SetText(status)
			
			balanceLabel.SetText(fmt.Sprintf("$%.2f", player.Balance))
		},
	)
	
	// Create scroll container with fixed height for players
	playersScroll := container.NewScroll(ui.playersList)
	playersScroll.SetMinSize(fyne.NewSize(500, 120)) // Increased height
	
	playersSection := container.NewVBox(
		widget.NewLabel("👥 Players"),
		playersScroll,
	)
	
	// Simple betting section - prominently displayed
	ui.betAmountEntry = widget.NewEntry()
	ui.betAmountEntry.SetPlaceHolder("Enter bet amount (e.g., 10)")
	ui.betAmountEntry.SetText("10") // Default bet amount
	ui.betAmountEntry.Validator = func(s string) error {
		if s == "" {
			return nil
		}
		amount, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return fmt.Errorf("invalid number")
		}
		if amount < ui.config.Game.MinBet || amount > ui.config.Game.MaxBet {
			return fmt.Errorf("bet must be between $%.2f and $%.2f", 
				ui.config.Game.MinBet, ui.config.Game.MaxBet)
		}
		return nil
	}
	
	// Large, prominent betting buttons
	ui.headsButton = widget.NewButton("👑 BET HEADS", func() {
		ui.placeBet(game.Heads)
	})
	ui.headsButton.Importance = widget.HighImportance
	
	ui.tailsButton = widget.NewButton("🦅 BET TAILS", func() {
		ui.placeBet(game.Tails)
	})
	ui.tailsButton.Importance = widget.HighImportance
	
	bettingSection := container.NewVBox(
		widget.NewLabel("💰 Place Your Bet"),
		ui.betAmountEntry,
		widget.NewSeparator(),
		ui.headsButton,
		ui.tailsButton,
	)
	
	// Game result
	ui.gameResult = widget.NewLabel("🎯 Connecting to multiplayer game...")
	ui.gameResult.Alignment = fyne.TextAlignCenter
	ui.gameResult.Wrapping = fyne.TextWrapWord
	
	// Game history section
	ui.historyList = widget.NewList(
		func() int { return len(ui.gameHistory) },
		func() fyne.CanvasObject {
			return container.NewHBox(
				widget.NewLabel("Round"),
				widget.NewLabel("Result"),
				widget.NewLabel("Winner"),
			)
		},
		func(id widget.ListItemID, item fyne.CanvasObject) {
			if id >= len(ui.gameHistory) {
				return
			}
			history := ui.gameHistory[id]
			cont := item.(*fyne.Container)
			
			roundLabel := cont.Objects[0].(*widget.Label)
			resultLabel := cont.Objects[1].(*widget.Label)
			winnerLabel := cont.Objects[2].(*widget.Label)
			
			roundLabel.SetText(fmt.Sprintf("#%d", len(ui.gameHistory)-id))
			
			coinEmoji := "👑"
			if history.CoinResult == game.Tails {
				coinEmoji = "🦅"
			}
			resultLabel.SetText(fmt.Sprintf("%s %s", coinEmoji, strings.ToUpper(history.CoinResult.String())))
			
			winnerText := "No winners"
			if len(history.Winners) > 0 {
				winnerText = fmt.Sprintf("%d winners", len(history.Winners))
			}
			winnerLabel.SetText(winnerText)
		},
	)
	
	// Create scroll container with fixed height for history
	historyScroll := container.NewScroll(ui.historyList)
	historyScroll.SetMinSize(fyne.NewSize(500, 150)) // Increased height
	
	historySection := container.NewVBox(
		widget.NewLabel("📊 Recent Games"),
		historyScroll,
	)
	
	// Player scoreboard section
	ui.scoreboardList = widget.NewList(
		func() int { return len(ui.playerStats) },
		func() fyne.CanvasObject {
			return container.NewHBox(
				widget.NewLabel("Player"),
				widget.NewLabel("Balance"),
				widget.NewLabel("W/L"),
				widget.NewLabel("Profit"),
			)
		},
		func(id widget.ListItemID, item fyne.CanvasObject) {
			// Convert map to slice for consistent ordering
			stats := make([]*PlayerStats, 0, len(ui.playerStats))
			for _, stat := range ui.playerStats {
				stats = append(stats, stat)
			}
			
			if id >= len(stats) {
				return
			}
			
			stat := stats[id]
			cont := item.(*fyne.Container)
			
			nameLabel := cont.Objects[0].(*widget.Label)
			balanceLabel := cont.Objects[1].(*widget.Label)
			wlLabel := cont.Objects[2].(*widget.Label)
			profitLabel := cont.Objects[3].(*widget.Label)
			
			nameLabel.SetText(stat.PlayerName)
			balanceLabel.SetText(fmt.Sprintf("$%.0f", stat.CurrentBalance))
			
			if stat.TotalGames > 0 {
				wlLabel.SetText(fmt.Sprintf("%d/%d", stat.GamesWon, stat.GamesLost))
				profitColor := "🟢"
				if stat.NetProfit < 0 {
					profitColor = "🔴"
				}
				profitLabel.SetText(fmt.Sprintf("%s$%.0f", profitColor, stat.NetProfit))
			} else {
				wlLabel.SetText("0/0")
				profitLabel.SetText("$0")
			}
		},
	)
	
	// Create scroll container with fixed height for scoreboard
	scoreboardScroll := container.NewScroll(ui.scoreboardList)
	scoreboardScroll.SetMinSize(fyne.NewSize(500, 150)) // Increased height
	
	scoreboardSection := container.NewVBox(
		widget.NewLabel("🏆 Scoreboard"),
		scoreboardScroll,
	)
	
	// Comprehensive layout with history and scoreboard
	mainPanel := container.NewVBox(
		statusSection,
		widget.NewSeparator(),
		timerSection,
		bettingSection,
		widget.NewSeparator(),
		ui.gameResult,
		widget.NewSeparator(),
		playersSection,
		widget.NewSeparator(),
		historySection,
		widget.NewSeparator(),
		scoreboardSection,
	)
	
	// Scroll container for smaller screens
	scrollContent := container.NewScroll(mainPanel)
	scrollContent.SetMinSize(fyne.NewSize(520, 900))
	
	ui.window.SetContent(scrollContent)
	ui.window.Resize(fyne.NewSize(580, 1000))
	
	// Auto-connect to server
	go func() {
		ui.connectToServer()
	}()
}

// connectToServer connects to the multiplayer server
func (ui *MultiplayerGameUI) connectToServer() {
	ui.updateConnectionStatus("🔄 Connecting...")
	
	go func() {
		if err := ui.networkClient.Connect(); err != nil {
			ui.logger.Error("Failed to connect", zap.Error(err))
			// Queue UI update to be executed on main thread
			ui.queueUIUpdate(func() {
				ui.connectionStatus.SetText("❌ Connection failed: " + err.Error())
			})
			return
		}
		
		// Queue UI update to be executed on main thread
		ui.queueUIUpdate(func() {
			ui.connectionStatus.SetText("✅ Connected")
		})
		
		// Auto-join default room if configured
		if ui.config.Multiplayer.AutoJoin && ui.config.Multiplayer.DefaultRoom != "" {
			time.Sleep(1 * time.Second) // Brief delay for connection to stabilize
			ui.joinRoom(ui.config.Multiplayer.DefaultRoom)
		}
	}()
}

// disconnectFromServer disconnects from the server
func (ui *MultiplayerGameUI) disconnectFromServer() {
	ui.networkClient.Disconnect()
	ui.queueUIUpdate(func() {
		ui.updateConnectionStatus("🔄 Disconnected")
		ui.roomInfo.SetText("Not in room")
		ui.currentPlayers = nil
	})
}

// joinRoom joins a multiplayer room
func (ui *MultiplayerGameUI) joinRoom(roomID string) {
	if !ui.networkClient.IsConnected() {
		dialog.ShowError(fmt.Errorf("not connected to server"), ui.window)
		return
	}
	
	go func() {
		if err := ui.networkClient.JoinRoom(roomID, ui.balance); err != nil {
			ui.logger.Error("Failed to join room", zap.Error(err))
			ui.queueUIUpdate(func() {
				dialog.ShowError(fmt.Errorf("failed to join room: %v", err), ui.window)
			})
			return
		}
		
		// Queue UI update to be executed on main thread
		ui.queueUIUpdate(func() {
			ui.roomInfo.SetText(fmt.Sprintf("📍 Room: %s", roomID))
		})
		ui.logger.Info("Joined room", zap.String("room_id", roomID))
	}()
}

// leaveRoom leaves the current room
func (ui *MultiplayerGameUI) leaveRoom() {
	go func() {
		if err := ui.networkClient.LeaveRoom(); err != nil {
			ui.logger.Error("Failed to leave room", zap.Error(err))
			return
		}
		
		// Queue UI update to be executed on main thread
		ui.queueUIUpdate(func() {
			ui.roomInfo.SetText("Not in room")
			ui.currentPlayers = nil
		})
		ui.logger.Info("Left room")
	}()
}

// placeBet places a bet in the multiplayer game
func (ui *MultiplayerGameUI) placeBet(choice game.Side) {
	if ui.networkClient.GetCurrentRoom() == "" {
		dialog.ShowInformation("No Room", "Join a room first", ui.window)
		return
	}
	
	if ui.gameState != network.StateBetting {
		dialog.ShowInformation("Betting Closed", "Betting phase is not active", ui.window)
		return
	}
	
	amountStr := ui.betAmountEntry.Text
	if amountStr == "" {
		dialog.ShowError(fmt.Errorf("enter bet amount"), ui.window)
		return
	}
	
	amount, err := strconv.ParseFloat(amountStr, 64)
	if err != nil {
		dialog.ShowError(fmt.Errorf("invalid bet amount"), ui.window)
		return
	}
	
	go func() {
		if err := ui.networkClient.PlaceBet(amount, choice); err != nil {
			ui.queueUIUpdate(func() {
				dialog.ShowError(fmt.Errorf("failed to place bet: %v", err), ui.window)
			})
			return
		}
		
		// Queue UI update to be executed on main thread
		ui.queueUIUpdate(func() {
			ui.updateBettingButtons()
			ui.gameResult.SetText(fmt.Sprintf("🎲 Bet placed: $%.2f on %s", amount, strings.ToUpper(choice.String())))
		})
	}()
}

// Message handlers

// handleRoomUpdate handles room state updates
func (ui *MultiplayerGameUI) handleRoomUpdate(msg *network.Message) {
	var roomUpdate network.RoomUpdateData
	if err := msg.GetData(&roomUpdate); err != nil {
		ui.logger.Error("Failed to parse room update", zap.Error(err))
		return
	}
	
	ui.currentPlayers = roomUpdate.Players
	ui.gameState = roomUpdate.GameState
	
	// Update local player balance from server state and track player stats
	for _, player := range roomUpdate.Players {
		if player.ID == ui.playerID {
			ui.balance = player.Balance
		}
		
		// Update or create player stats
		if ui.playerStats[player.ID] == nil {
			ui.playerStats[player.ID] = &PlayerStats{
				PlayerName:     player.Name,
				CurrentBalance: player.Balance,
				LastSeen:       time.Now(),
			}
		} else {
			ui.playerStats[player.ID].PlayerName = player.Name
			ui.playerStats[player.ID].CurrentBalance = player.Balance
			ui.playerStats[player.ID].LastSeen = time.Now()
		}
	}
	
	// Queue UI updates to be executed on main thread
	ui.queueUIUpdate(func() {
		playerCount := len(roomUpdate.Players)
		ui.roomInfo.SetText(fmt.Sprintf("📍 Room: %s (%d/%d players)", 
			roomUpdate.RoomID, playerCount, roomUpdate.MaxPlayers))
		ui.updateBettingButtons()
		ui.historyList.Refresh()
		ui.scoreboardList.Refresh()
	})
}

// handleTimerUpdate handles timer updates
func (ui *MultiplayerGameUI) handleTimerUpdate(msg *network.Message) {
	var timerData network.TimerData
	if err := msg.GetData(&timerData); err != nil {
		ui.logger.Error("Failed to parse timer update", zap.Error(err))
		return
	}
	
	ui.timerSeconds = timerData.SecondsLeft
	ui.totalSeconds = timerData.TotalSeconds
	
	// Queue UI updates to be executed on main thread
	ui.queueUIUpdate(func() {
		// Update timer display
		minutes := timerData.SecondsLeft / 60
		seconds := timerData.SecondsLeft % 60
		ui.timerLabel.SetText(fmt.Sprintf("⏱️ %s: %d:%02d", 
			strings.Title(string(timerData.Phase)), minutes, seconds))
		
		// Update progress bar
		if timerData.TotalSeconds > 0 {
			progress := float64(timerData.TotalSeconds-timerData.SecondsLeft) / float64(timerData.TotalSeconds)
			ui.progressBar.SetValue(progress)
		}
	})
}

// handleGameResult handles game result announcements
func (ui *MultiplayerGameUI) handleGameResult(msg *network.Message) {
	var result network.GameResultData
	if err := msg.GetData(&result); err != nil {
		ui.logger.Error("Failed to parse game result", zap.Error(err))
		return
	}
	
	// Add to history
	ui.gameHistory = append([]*network.GameResultData{&result}, ui.gameHistory...)
	if len(ui.gameHistory) > 10 {
		ui.gameHistory = ui.gameHistory[:10]
	}
	
	// Update player statistics for all participants
	ui.updatePlayerStatistics(&result)
	
	// Display result
	coinEmoji := "👑"
	if result.CoinResult == game.Tails {
		coinEmoji = "🦅"
	}
	
	resultText := fmt.Sprintf("%s %s", coinEmoji, strings.ToUpper(result.CoinResult.String()))
	
	// Check if we won
	var playerResult *network.PlayerResult
	for _, winner := range result.Winners {
		if winner.PlayerID == ui.playerID {
			playerResult = &winner
			break
		}
	}
	if playerResult == nil {
		for _, loser := range result.Losers {
			if loser.PlayerID == ui.playerID {
				playerResult = &loser
				break
			}
		}
	}
	
	// Queue UI updates to be executed on main thread
	ui.queueUIUpdate(func() {
		if playerResult != nil {
			ui.balance = playerResult.NewBalance
			if playerResult.Won {
				ui.gameResult.SetText(fmt.Sprintf("🎉 %s - You won $%.2f!", 
					resultText, playerResult.Payout))
			} else {
				ui.gameResult.SetText(fmt.Sprintf("😞 %s - You lost $%.2f", 
					resultText, playerResult.Bet.Amount))
			}
		} else {
			ui.gameResult.SetText(fmt.Sprintf("🎲 %s (You didn't bet)", resultText))
		}
		
		ui.updateBettingButtons()
		ui.historyList.Refresh()
		ui.scoreboardList.Refresh()
	})
}

// handleBetPhase handles betting phase start
func (ui *MultiplayerGameUI) handleBetPhase(msg *network.Message) {
	ui.gameState = network.StateBetting
	
	// Queue UI updates to be executed on main thread
	ui.queueUIUpdate(func() {
		ui.updateBettingButtons()
		ui.gameResult.SetText("🎲 Betting phase started! Place your bets!")
	})
}

// handleError handles error messages
func (ui *MultiplayerGameUI) handleError(msg *network.Message) {
	var errorData network.ErrorData
	if err := msg.GetData(&errorData); err != nil {
		ui.logger.Error("Failed to parse error message", zap.Error(err))
		return
	}
	
	// Queue UI updates to be executed on main thread
	ui.queueUIUpdate(func() {
		dialog.ShowError(fmt.Errorf("%s: %s", errorData.Code, errorData.Message), ui.window)
	})
}

// Helper methods

// updateConnectionStatus updates the connection status label
func (ui *MultiplayerGameUI) updateConnectionStatus(status string) {
	// Ensure UI updates happen on the main thread
	ui.connectionStatus.SetText(status)
}

// updateBettingButtons enables/disables betting buttons based on game state
func (ui *MultiplayerGameUI) updateBettingButtons() {
	inRoom := ui.networkClient.GetCurrentRoom() != ""
	validAmount := ui.betAmountEntry.Validate() == nil && ui.betAmountEntry.Text != ""
	bettingActive := ui.gameState == network.StateBetting
	
	// Enable betting if in room, amount is valid, and betting is active
	canBet := inRoom && validAmount && bettingActive
	
	if canBet {
		ui.headsButton.Enable()
		ui.tailsButton.Enable()
		ui.headsButton.SetText("👑 BET HEADS")
		ui.tailsButton.SetText("🦅 BET TAILS")
	} else {
		ui.headsButton.Disable()
		ui.tailsButton.Disable()
		
		// Show helpful messages on buttons
		if !inRoom {
			ui.headsButton.SetText("👑 (Join room first)")
			ui.tailsButton.SetText("🦅 (Join room first)")
		} else if !validAmount {
			ui.headsButton.SetText("👑 (Enter bet amount)")
			ui.tailsButton.SetText("🦅 (Enter bet amount)")
		} else if !bettingActive {
			ui.headsButton.SetText("👑 (Waiting for round)")
			ui.tailsButton.SetText("🦅 (Waiting for round)")
		}
	}
	
	// Debug logging
	ui.logger.Info("Betting buttons updated",
		zap.Bool("in_room", inRoom),
		zap.Bool("valid_amount", validAmount),
		zap.Bool("betting_active", bettingActive),
		zap.String("game_state", string(ui.gameState)),
		zap.Bool("can_bet", canBet),
	)
}

// updatePlayerStatistics updates player statistics based on game results
func (ui *MultiplayerGameUI) updatePlayerStatistics(result *network.GameResultData) {
	// Process winners
	for _, winner := range result.Winners {
		if ui.playerStats[winner.PlayerID] == nil {
			ui.playerStats[winner.PlayerID] = &PlayerStats{
				PlayerName: fmt.Sprintf("Player%s", winner.PlayerID[len(winner.PlayerID)-4:]),
			}
		}
		
		stats := ui.playerStats[winner.PlayerID]
		stats.TotalGames++
		stats.GamesWon++
		stats.TotalBet += winner.Bet.Amount
		stats.TotalWon += winner.Payout
		stats.NetProfit += (winner.Payout - winner.Bet.Amount)
		stats.CurrentBalance = winner.NewBalance
		stats.LastSeen = time.Now()
	}
	
	// Process losers
	for _, loser := range result.Losers {
		if ui.playerStats[loser.PlayerID] == nil {
			ui.playerStats[loser.PlayerID] = &PlayerStats{
				PlayerName: fmt.Sprintf("Player%s", loser.PlayerID[len(loser.PlayerID)-4:]),
			}
		}
		
		stats := ui.playerStats[loser.PlayerID]
		stats.TotalGames++
		stats.GamesLost++
		stats.TotalBet += loser.Bet.Amount
		stats.NetProfit -= loser.Bet.Amount
		stats.CurrentBalance = loser.NewBalance
		stats.LastSeen = time.Now()
	}
}
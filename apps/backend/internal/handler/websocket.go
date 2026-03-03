package handler

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/grafikarsa/backend/internal/dto"
	"github.com/grafikarsa/backend/internal/middleware"
)

// ============================================================================
// WEBSOCKET HUB
// ============================================================================

// Client represents a connected WebSocket client
type Client struct {
	Conn     *websocket.Conn
	UserID   uuid.UUID
	Username string
	Send     chan []byte
}

// Hub maintains the set of active clients and broadcasts messages
type Hub struct {
	// Registered clients by user ID
	clients map[uuid.UUID]*Client

	// Register requests from clients
	register chan *Client

	// Unregister requests from clients
	unregister chan *Client

	// Broadcast to specific users
	broadcast chan *BroadcastMessage

	// Mutex for thread safety
	mu sync.RWMutex
}

// BroadcastMessage represents a message to send to specific users
type BroadcastMessage struct {
	UserIDs []uuid.UUID
	Event   dto.WSEvent
}

// NewHub creates a new WebSocket hub
func NewHub() *Hub {
	return &Hub{
		clients:    make(map[uuid.UUID]*Client),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan *BroadcastMessage, 256),
	}
}

// Run starts the hub's event loop
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			// Close existing connection if any
			if existing, ok := h.clients[client.UserID]; ok {
				close(existing.Send)
			}
			h.clients[client.UserID] = client
			h.mu.Unlock()

			// Broadcast online presence
			h.BroadcastPresence(client.UserID, true)
			log.Printf("[WS] User %s connected", client.Username)

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client.UserID]; ok {
				delete(h.clients, client.UserID)
				close(client.Send)
			}
			h.mu.Unlock()

			// Broadcast offline presence
			h.BroadcastPresence(client.UserID, false)
			log.Printf("[WS] User %s disconnected", client.Username)

		case msg := <-h.broadcast:
			data, err := json.Marshal(msg.Event)
			if err != nil {
				log.Printf("[WS] Error marshaling event: %v", err)
				continue
			}

			h.mu.RLock()
			for _, userID := range msg.UserIDs {
				if client, ok := h.clients[userID]; ok {
					select {
					case client.Send <- data:
					default:
						// Buffer full, skip
					}
				}
			}
			h.mu.RUnlock()
		}
	}
}

// BroadcastPresence broadcasts user presence to all connected clients
func (h *Hub) BroadcastPresence(userID uuid.UUID, isOnline bool) {
	event := dto.WSEvent{
		Type: "presence",
		Payload: dto.WSPresence{
			UserID:   userID,
			IsOnline: isOnline,
		},
	}

	data, err := json.Marshal(event)
	if err != nil {
		return
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	for _, client := range h.clients {
		select {
		case client.Send <- data:
		default:
		}
	}
}

// SendToUsers sends an event to specific users
func (h *Hub) SendToUsers(userIDs []uuid.UUID, event dto.WSEvent) {
	h.broadcast <- &BroadcastMessage{
		UserIDs: userIDs,
		Event:   event,
	}
}

// IsUserOnline checks if a user is currently connected
func (h *Hub) IsUserOnline(userID uuid.UUID) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	_, ok := h.clients[userID]
	return ok
}

// GetOnlineUsers returns a list of online user IDs
func (h *Hub) GetOnlineUsers() []uuid.UUID {
	h.mu.RLock()
	defer h.mu.RUnlock()

	users := make([]uuid.UUID, 0, len(h.clients))
	for userID := range h.clients {
		users = append(users, userID)
	}
	return users
}

// ============================================================================
// WEBSOCKET HANDLER
// ============================================================================

type WebSocketHandler struct {
	Hub *Hub
}

func NewWebSocketHandler() *WebSocketHandler {
	hub := NewHub()
	go hub.Run()

	return &WebSocketHandler{Hub: hub}
}

// HandleWebSocket handles WebSocket connections
func (h *WebSocketHandler) HandleWebSocket(c *websocket.Conn) {
	// Get user info from locals (set by auth middleware before upgrade)
	userID, ok := c.Locals("user_id").(uuid.UUID)
	if !ok {
		c.Close()
		return
	}

	username, _ := c.Locals("username").(string)

	client := &Client{
		Conn:     c,
		UserID:   userID,
		Username: username,
		Send:     make(chan []byte, 256),
	}

	h.Hub.register <- client

	// Start goroutines for reading and writing
	go h.writePump(client)
	h.readPump(client)
}

// readPump pumps messages from the WebSocket connection to the hub
func (h *WebSocketHandler) readPump(client *Client) {
	defer func() {
		h.Hub.unregister <- client
		client.Conn.Close()
	}()

	for {
		_, message, err := client.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("[WS] Error reading message: %v", err)
			}
			break
		}

		// Parse the incoming message
		var msg struct {
			Type    string          `json:"type"`
			Payload json.RawMessage `json:"payload"`
		}

		if err := json.Unmarshal(message, &msg); err != nil {
			continue
		}

		// Handle different message types
		switch msg.Type {
		case "typing":
			h.handleTyping(client, msg.Payload)
		case "ping":
			// Respond with pong
			pong := dto.WSEvent{Type: "pong", Payload: nil}
			data, _ := json.Marshal(pong)
			client.Send <- data
		}
	}
}

// writePump pumps messages from the hub to the WebSocket connection
func (h *WebSocketHandler) writePump(client *Client) {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		client.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-client.Send:
			if !ok {
				// Channel closed
				client.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := client.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}

		case <-ticker.C:
			// Send ping to keep connection alive
			if err := client.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// handleTyping handles typing indicator events
func (h *WebSocketHandler) handleTyping(client *Client, payload json.RawMessage) {
	var typingPayload struct {
		ConversationID uuid.UUID `json:"conversation_id"`
		IsTyping       bool      `json:"is_typing"`
	}

	if err := json.Unmarshal(payload, &typingPayload); err != nil {
		return
	}

	// Broadcast typing event to other participants
	// Note: In production, you'd verify the user is in the conversation
	event := dto.WSEvent{
		Type: "typing",
		Payload: dto.WSTyping{
			ConversationID: typingPayload.ConversationID,
			UserID:         client.UserID,
			Username:       client.Username,
			IsTyping:       typingPayload.IsTyping,
		},
	}

	// TODO: Get conversation participants and send only to them
	// For now, this is a placeholder that would need conversation lookup
	h.Hub.SendToUsers([]uuid.UUID{}, event)
}

// ============================================================================
// HELPER METHODS FOR DM HANDLER INTEGRATION
// ============================================================================

// BroadcastNewMessage broadcasts a new message to conversation participants
func (h *WebSocketHandler) BroadcastNewMessage(participantIDs []uuid.UUID, convID uuid.UUID, msg dto.MessageResponse) {
	event := dto.WSEvent{
		Type: "message.new",
		Payload: dto.WSMessageNew{
			ConversationID: convID,
			Message:        msg,
		},
	}
	h.Hub.SendToUsers(participantIDs, event)
}

// BroadcastMessageDeleted broadcasts a deleted message event
func (h *WebSocketHandler) BroadcastMessageDeleted(participantIDs []uuid.UUID, convID, msgID uuid.UUID) {
	event := dto.WSEvent{
		Type: "message.deleted",
		Payload: dto.WSMessageDeleted{
			ConversationID: convID,
			MessageID:      msgID,
		},
	}
	h.Hub.SendToUsers(participantIDs, event)
}

// BroadcastReaction broadcasts a reaction event
func (h *WebSocketHandler) BroadcastReaction(participantIDs []uuid.UUID, convID, msgID uuid.UUID, reaction dto.ReactionResponse, action string) {
	event := dto.WSEvent{
		Type: "message.reaction",
		Payload: dto.WSMessageReaction{
			ConversationID: convID,
			MessageID:      msgID,
			Reaction:       reaction,
			Action:         action,
		},
	}
	h.Hub.SendToUsers(participantIDs, event)
}

// BroadcastReadReceipt broadcasts a read receipt event
func (h *WebSocketHandler) BroadcastReadReceipt(participantIDs []uuid.UUID, convID, userID uuid.UUID) {
	event := dto.WSEvent{
		Type: "read.receipt",
		Payload: dto.WSReadReceipt{
			ConversationID: convID,
			UserID:         userID,
			ReadAt:         time.Now(),
		},
	}
	h.Hub.SendToUsers(participantIDs, event)
}

// ============================================================================
// FIBER UPGRADE HANDLER
// ============================================================================

// WebSocketUpgrade is a middleware to upgrade HTTP connections to WebSocket
func (h *WebSocketHandler) WebSocketUpgrade(authMiddleware *middleware.AuthMiddleware) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Check if it's a WebSocket upgrade request
		if websocket.IsWebSocketUpgrade(c) {
			// Authenticate the user first
			token := c.Query("token")
			if token == "" {
				return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
					"success": false,
					"error": fiber.Map{
						"code":    "UNAUTHORIZED",
						"message": "Token diperlukan untuk WebSocket",
					},
				})
			}

			// Verify token and get user info
			claims, err := authMiddleware.GetJWTService().ValidateAccessToken(token)
			if err != nil {
				return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
					"success": false,
					"error": fiber.Map{
						"code":    "INVALID_TOKEN",
						"message": "Token tidak valid",
					},
				})
			}

			// Set user info in locals for WebSocket handler
			userID, _ := uuid.Parse(claims.Sub)
			c.Locals("user_id", userID)
			// Username will be loaded later if needed, JWT only has user ID
			c.Locals("username", "")

			return c.Next()
		}

		return fiber.ErrUpgradeRequired
	}
}

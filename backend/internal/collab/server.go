package collab

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/collab-docs/backend/internal/auth"
	"github.com/collab-docs/backend/internal/db"
	"github.com/collab-docs/backend/internal/models"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

const (
	// Time allowed to write a message to the peer
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer
	pongWait = 60 * time.Second

	// Send pings to peer with this period (must be less than pongWait)
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer
	maxMessageSize = 512 * 1024 // 512KB
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// Allow all origins for development
		// In production, validate against allowed origins
		return true
	},
}

// Server handles WebSocket connections for collaboration
type Server struct {
	manager *RoomManager
	db      *db.DB
}

// NewServer creates a new collaboration server
func NewServer(manager *RoomManager, database *db.DB) *Server {
	return &Server{
		manager: manager,
		db:      database,
	}
}

// HandleWebSocket handles WebSocket upgrade and connection
func (s *Server) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Get document ID from path
	docIDStr := r.PathValue("docId")
	if docIDStr == "" {
		// Fallback for mux implementations without path variables
		docIDStr = strings.TrimPrefix(r.URL.Path, "/collab/")
		docIDStr = strings.Trim(docIDStr, "/")
	}
	if docIDStr == "" {
		// Try query parameter as fallback
		docIDStr = r.URL.Query().Get("docId")
	}

	docID, err := uuid.Parse(docIDStr)
	if err != nil {
		http.Error(w, "Invalid document ID", http.StatusBadRequest)
		return
	}

	// Authenticate user
	user, permission, err := s.authenticateRequest(r, docID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	// Check if document exists
	doc, err := s.db.GetDocument(r.Context(), docID)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	if doc == nil {
		http.Error(w, "Document not found", http.StatusNotFound)
		return
	}

	// Upgrade to WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Failed to upgrade WebSocket: %v", err)
		return
	}

	// Create client
	client := NewClient(conn, user, permission)

	// Get or create room
	room, err := s.manager.GetOrCreateRoom(r.Context(), docID)
	if err != nil {
		log.Printf("Failed to get/create room: %v", err)
		conn.Close()
		return
	}

	// Register client
	room.Register(client)

	// Send connected message
	client.WriteJSON(map[string]interface{}{
		"type":       models.MsgTypeConnected,
		"userId":     user.ID.String(),
		"permission": permission,
		"docId":      docID.String(),
	})

	// Start client goroutines
	go s.writePump(client)
	go s.readPump(client, room)
}

// authenticateRequest authenticates the WebSocket request
func (s *Server) authenticateRequest(r *http.Request, docID uuid.UUID) (*models.User, string, error) {
	ctx := r.Context()

	// Try JWT token from query parameter
	token := r.URL.Query().Get("token")
	if token != "" {
		claims, err := auth.ValidateToken(token)
		if err == nil {
			userID, _ := uuid.Parse(claims.UserID)
			user, err := s.db.GetUser(ctx, userID)
			if err != nil || user == nil {
				return nil, "", err
			}

			perm, err := s.db.GetPermission(ctx, docID, userID)
			if err != nil || perm == nil {
				return nil, "", err
			}

			return user, perm.Role, nil
		}
	}

	// Try X-User-ID header for development
	userIDStr := r.Header.Get("X-User-ID")
	if userIDStr == "" {
		userIDStr = r.URL.Query().Get("userId")
	}
	if userIDStr == "" {
		// Default to Alice for development
		userIDStr = "11111111-1111-1111-1111-111111111111"
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return nil, "", err
	}

	user, err := s.db.GetUser(ctx, userID)
	if err != nil || user == nil {
		return nil, "", err
	}

	perm, err := s.db.GetPermission(ctx, docID, userID)
	if err != nil {
		return nil, "", err
	}

	// If no permission found, allow for development
	role := "view"
	if perm != nil {
		role = perm.Role
	}

	return user, role, nil
}

// readPump reads messages from the WebSocket connection
func (s *Server) readPump(client *Client, room *Room) {
	defer func() {
		room.Unregister(client)
		client.Conn.Close()
	}()

	client.Conn.SetReadLimit(maxMessageSize)
	client.Conn.SetReadDeadline(time.Now().Add(pongWait))
	client.Conn.SetPongHandler(func(string) error {
		client.Conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		messageType, message, err := client.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		if messageType == websocket.TextMessage {
			s.handleTextMessage(client, room, message)
		} else if messageType == websocket.BinaryMessage {
			s.handleBinaryMessage(client, room, message)
		}
	}
}

// handleTextMessage handles JSON text messages
func (s *Server) handleTextMessage(client *Client, room *Room, message []byte) {
	var msg struct {
		Type    string          `json:"type"`
		Payload json.RawMessage `json:"payload"`
	}

	if err := json.Unmarshal(message, &msg); err != nil {
		return
	}

	switch msg.Type {
	case "presence":
		var presence models.Presence
		if err := json.Unmarshal(msg.Payload, &presence); err != nil {
			return
		}
		presence.UserID = client.UserID.String()
		presence.Name = client.User.Name
		room.UpdatePresence(client, &presence)

	case "update":
		var updateMsg struct {
			Update []byte `json:"update"`
		}
		if err := json.Unmarshal(msg.Payload, &updateMsg); err != nil {
			return
		}
		room.ApplyUpdate(client, updateMsg.Update)
	}
}

// handleBinaryMessage handles binary CRDT updates (Yjs protocol)
func (s *Server) handleBinaryMessage(client *Client, room *Room, message []byte) {
	// Check permission
	if !client.CanEdit() {
		return // Silently ignore updates from non-editors
	}

	// Store the update for persistence
	room.Doc.ApplyUpdate(message)

	// Broadcast raw binary message to all other clients directly
	// This is essential for y-websocket compatibility
	room.BroadcastBinary(client, message)
}

// writePump writes messages to the WebSocket connection
func (s *Server) writePump(client *Client) {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		client.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-client.Send:
			client.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// Channel closed
				client.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := client.Conn.WriteMessage(websocket.BinaryMessage, message); err != nil {
				return
			}

		case <-ticker.C:
			client.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := client.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// ServeHTTP implements http.Handler
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.HandleWebSocket(w, r)
}

// CreateHandler creates an HTTP handler for WebSocket connections
func (s *Server) CreateHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s.HandleWebSocket(w, r)
	}
}

// RoomStats returns statistics about active rooms
func (s *Server) RoomStats(ctx context.Context) map[string]interface{} {
	return map[string]interface{}{
		"roomCount": s.manager.RoomCount(),
	}
}

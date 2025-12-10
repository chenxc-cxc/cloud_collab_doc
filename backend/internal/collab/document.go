package collab

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/collab-docs/backend/internal/models"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// Document represents a collaborative CRDT document
// This is a simplified Yjs-compatible implementation
type Document struct {
	ID         uuid.UUID
	content    []byte
	updates    [][]byte
	version    uint64
	mu         sync.RWMutex
	lastUpdate time.Time
}

// NewDocument creates a new empty document
func NewDocument(id uuid.UUID) *Document {
	return &Document{
		ID:         id,
		content:    []byte{},
		updates:    make([][]byte, 0),
		version:    0,
		lastUpdate: time.Now(),
	}
}

// LoadFromSnapshot loads document state from a snapshot
func (d *Document) LoadFromSnapshot(snapshot []byte, version uint64) {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Try to parse as JSON (our format)
	var state struct {
		Content []byte   `json:"content"`
		Updates [][]byte `json:"updates"`
		Version uint64   `json:"version"`
	}
	if err := json.Unmarshal(snapshot, &state); err == nil {
		d.content = state.Content
		d.updates = state.Updates
		d.version = state.Version
	} else {
		// Fallback: treat as raw content
		d.content = snapshot
		d.version = version
		d.updates = make([][]byte, 0)
	}
	d.lastUpdate = time.Now()
}

// ApplyUpdate applies a binary update to the document
func (d *Document) ApplyUpdate(update []byte) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Store the update
	d.updates = append(d.updates, update)
	d.version++
	d.lastUpdate = time.Now()

	return nil
}

// GetState returns the current document state (snapshot + pending updates)
func (d *Document) GetState() ([]byte, [][]byte) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	return d.content, d.updates
}

// GetSnapshot returns a compacted snapshot of the document
func (d *Document) GetSnapshot() []byte {
	d.mu.RLock()
	defer d.mu.RUnlock()

	// In a real implementation, this would compact all updates into a single snapshot
	// For now, we'll serialize the content + updates
	state := struct {
		Content []byte   `json:"content"`
		Updates [][]byte `json:"updates"`
		Version uint64   `json:"version"`
	}{
		Content: d.content,
		Updates: d.updates,
		Version: d.version,
	}

	data, _ := json.Marshal(state)
	return data
}

// GetVersion returns the current version
func (d *Document) GetVersion() uint64 {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.version
}

// LastUpdate returns the time of the last update
func (d *Document) LastUpdate() time.Time {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.lastUpdate
}

// Client represents a connected WebSocket client
type Client struct {
	ID         string
	UserID     uuid.UUID
	User       *models.User
	Permission string
	Conn       *websocket.Conn
	Room       *Room
	Send       chan []byte
	mu         sync.Mutex
}

// NewClient creates a new client
func NewClient(conn *websocket.Conn, user *models.User, permission string) *Client {
	return &Client{
		ID:         uuid.New().String(),
		UserID:     user.ID,
		User:       user,
		Permission: permission,
		Conn:       conn,
		Send:       make(chan []byte, 256),
	}
}

// CanEdit returns true if the client can edit the document
func (c *Client) CanEdit() bool {
	return c.Permission == models.RoleOwner || c.Permission == models.RoleEdit
}

// WriteMessage sends a message to the client
func (c *Client) WriteMessage(data []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.Conn.WriteMessage(websocket.BinaryMessage, data)
}

// WriteJSON sends a JSON message to the client
func (c *Client) WriteJSON(v interface{}) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.Conn.WriteJSON(v)
}

// Close closes the client connection
func (c *Client) Close() {
	close(c.Send)
	c.Conn.Close()
}

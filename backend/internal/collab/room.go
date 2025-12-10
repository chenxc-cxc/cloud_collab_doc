package collab

import (
	"context"
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/collab-docs/backend/internal/db"
	"github.com/collab-docs/backend/internal/models"
	"github.com/collab-docs/backend/internal/redis"
	"github.com/google/uuid"
)

// Room represents a collaboration room for a document
type Room struct {
	ID           uuid.UUID
	Doc          *Document
	clients      map[string]*Client
	presence     map[string]*models.Presence
	mu           sync.RWMutex
	broadcast    chan *BroadcastMessage
	register     chan *Client
	unregister   chan *Client
	pubsub       *redis.PubSub
	db           *db.DB
	instanceID   string
	lastActivity time.Time
	ctx          context.Context
	cancel       context.CancelFunc
}

// BroadcastMessage represents a message to broadcast
type BroadcastMessage struct {
	Type    string
	Data    []byte
	Sender  *Client
	SkipIDs map[string]bool
}

// NewRoom creates a new collaboration room
func NewRoom(ctx context.Context, docID uuid.UUID, pubsub *redis.PubSub, database *db.DB, instanceID string) *Room {
	roomCtx, cancel := context.WithCancel(ctx)

	room := &Room{
		ID:           docID,
		Doc:          NewDocument(docID),
		clients:      make(map[string]*Client),
		presence:     make(map[string]*models.Presence),
		broadcast:    make(chan *BroadcastMessage, 256),
		register:     make(chan *Client),
		unregister:   make(chan *Client),
		pubsub:       pubsub,
		db:           database,
		instanceID:   instanceID,
		lastActivity: time.Now(),
		ctx:          roomCtx,
		cancel:       cancel,
	}

	return room
}

// Run starts the room's main loop
func (r *Room) Run() {
	// Subscribe to Redis channel for cross-instance sync
	roomChannel := redis.GetRoomChannel(r.ID.String())
	presenceChannel := redis.GetPresenceChannel(r.ID.String())

	r.pubsub.Subscribe(roomChannel, r.handleRedisMessage)
	r.pubsub.Subscribe(presenceChannel, r.handlePresenceMessage)

	// Start idle timer (check every 30 seconds)
	idleTimer := time.NewTicker(30 * time.Second)
	defer idleTimer.Stop()

	// Auto-save timer (save every 5 seconds if there are updates)
	saveTimer := time.NewTicker(5 * time.Second)
	defer saveTimer.Stop()

	var lastSavedVersion uint64 = r.Doc.GetVersion()

	for {
		select {
		case <-r.ctx.Done():
			r.cleanup()
			return

		case client := <-r.register:
			r.handleRegister(client)

		case client := <-r.unregister:
			r.handleUnregister(client)

		case msg := <-r.broadcast:
			r.handleBroadcast(msg)

		case <-saveTimer.C:
			// Auto-save if there are new updates
			currentVersion := r.Doc.GetVersion()
			if currentVersion > lastSavedVersion {
				go r.saveSnapshot()
				lastSavedVersion = currentVersion
			}

		case <-idleTimer.C:
			r.checkIdle()
		}
	}
}

// handleRegister registers a new client
func (r *Room) handleRegister(client *Client) {
	r.mu.Lock()
	r.clients[client.ID] = client
	client.Room = r
	r.lastActivity = time.Now()
	r.mu.Unlock()

	// Send current document state to new client
	r.sendSyncState(client)

	// Send current presence to new client
	r.sendPresenceState(client)

	log.Printf("Client %s joined room %s (total: %d)", client.ID, r.ID, len(r.clients))
}

// handleUnregister removes a client
func (r *Room) handleUnregister(client *Client) {
	r.mu.Lock()
	if _, ok := r.clients[client.ID]; ok {
		delete(r.clients, client.ID)
		delete(r.presence, client.UserID.String())
		r.lastActivity = time.Now()
	}
	clientCount := len(r.clients)
	r.mu.Unlock()

	// Broadcast presence removal
	r.broadcastPresenceUpdate(client.UserID.String(), nil)

	log.Printf("Client %s left room %s (total: %d)", client.ID, r.ID, clientCount)

	// Save snapshot immediately when last client leaves
	if clientCount == 0 && r.Doc.GetVersion() > 0 {
		go r.saveSnapshot()
	}
}

// handleBroadcast broadcasts a message to all clients
func (r *Room) handleBroadcast(msg *BroadcastMessage) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, client := range r.clients {
		if msg.SkipIDs != nil && msg.SkipIDs[client.ID] {
			continue
		}

		select {
		case client.Send <- msg.Data:
		default:
			// Client buffer full, skip
		}
	}
}

// handleRedisMessage handles messages from Redis pub/sub
func (r *Room) handleRedisMessage(channel string, payload []byte) {
	var msg redis.Message
	if err := json.Unmarshal(payload, &msg); err != nil {
		return
	}

	// Skip messages from this instance
	if msg.From == r.instanceID {
		return
	}

	// For yjs-sync messages, broadcast raw binary payload directly
	if msg.Type == "yjs-sync" {
		r.broadcastToClients(msg.Payload, nil)
		return
	}

	// For other message types, broadcast the full payload
	r.broadcast <- &BroadcastMessage{
		Type: msg.Type,
		Data: msg.Payload,
	}
}

// handlePresenceMessage handles presence updates from Redis
func (r *Room) handlePresenceMessage(channel string, payload []byte) {
	var msg redis.Message
	if err := json.Unmarshal(payload, &msg); err != nil {
		return
	}

	if msg.From == r.instanceID {
		return
	}

	var presence models.Presence
	if err := json.Unmarshal(msg.Payload, &presence); err != nil {
		return
	}

	// Update local presence and broadcast
	r.mu.Lock()
	r.presence[presence.UserID] = &presence
	r.mu.Unlock()

	r.broadcastToClients(msg.Payload, nil)
}

// y-websocket protocol message types
const (
	msgSync       = 0
	msgAwareness  = 1
	msgSyncStep1  = 0
	msgSyncStep2  = 1
	msgSyncUpdate = 2
)

// sendSyncState sends the current document state to a client using y-websocket protocol
func (r *Room) sendSyncState(client *Client) {
	_, updates := r.Doc.GetState()

	// Merge all updates into a single state for sync step 2
	// y-websocket protocol: [msgSync, msgSyncStep2, ...encodedUpdate]
	for _, update := range updates {
		// Create sync step 2 message: msgSync(0) + msgSyncStep2(1) + update data
		msg := make([]byte, 0, 2+len(update))
		msg = append(msg, msgSync)      // message type: sync
		msg = append(msg, msgSyncStep2) // sync step: step 2 (send update)
		msg = append(msg, update...)    // the actual update data

		select {
		case client.Send <- msg:
		default:
			// Buffer full, skip
		}
	}

	log.Printf("Sent %d updates to client %s for sync", len(updates), client.ID)
}

// sendPresenceState sends current presence to a client
func (r *Room) sendPresenceState(client *Client) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, p := range r.presence {
		data, _ := json.Marshal(struct {
			Type     string           `json:"type"`
			Presence *models.Presence `json:"presence"`
		}{
			Type:     models.MsgTypePresence,
			Presence: p,
		})
		client.WriteMessage(data)
	}
}

// broadcastPresenceUpdate broadcasts a presence update
func (r *Room) broadcastPresenceUpdate(userID string, presence *models.Presence) {
	msg := struct {
		Type     string           `json:"type"`
		UserID   string           `json:"userId"`
		Presence *models.Presence `json:"presence"`
	}{
		Type:     models.MsgTypePresence,
		UserID:   userID,
		Presence: presence,
	}

	data, _ := json.Marshal(msg)

	// Broadcast locally
	r.broadcastToClients(data, nil)

	// Publish to Redis for cross-instance sync
	r.pubsub.Publish(redis.GetPresenceChannel(r.ID.String()), &redis.Message{
		Type:    models.MsgTypePresence,
		From:    r.instanceID,
		Payload: data,
	})
}

// ApplyUpdate applies an update from a client
func (r *Room) ApplyUpdate(client *Client, update []byte) error {
	// Check permission
	if !client.CanEdit() {
		return nil // Silently ignore updates from non-editors
	}

	// Apply to document
	if err := r.Doc.ApplyUpdate(update); err != nil {
		return err
	}

	r.mu.Lock()
	r.lastActivity = time.Now()
	r.mu.Unlock()

	// Broadcast to other local clients
	msg := struct {
		Type   string `json:"type"`
		Update []byte `json:"update"`
	}{
		Type:   models.MsgTypeUpdate,
		Update: update,
	}
	data, _ := json.Marshal(msg)

	r.broadcastToClients(data, map[string]bool{client.ID: true})

	// Publish to Redis for cross-instance sync
	r.pubsub.Publish(redis.GetRoomChannel(r.ID.String()), &redis.Message{
		Type:    models.MsgTypeUpdate,
		From:    r.instanceID,
		Payload: data,
	})

	return nil
}

// UpdatePresence updates a client's presence
func (r *Room) UpdatePresence(client *Client, presence *models.Presence) {
	r.mu.Lock()
	r.presence[client.UserID.String()] = presence
	r.lastActivity = time.Now()
	r.mu.Unlock()

	r.broadcastPresenceUpdate(client.UserID.String(), presence)
}

// broadcastToClients broadcasts data to all local clients
func (r *Room) broadcastToClients(data []byte, skipIDs map[string]bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, client := range r.clients {
		if skipIDs != nil && skipIDs[client.ID] {
			continue
		}

		select {
		case client.Send <- data:
		default:
		}
	}
}

// BroadcastBinary broadcasts raw binary data to all clients except sender
// This is used for Yjs sync messages which should be relayed as-is
func (r *Room) BroadcastBinary(sender *Client, data []byte) {
	r.mu.Lock()
	r.lastActivity = time.Now()
	r.mu.Unlock()

	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, client := range r.clients {
		if client.ID == sender.ID {
			continue
		}

		select {
		case client.Send <- data:
		default:
			// Client buffer full, skip
		}
	}

	// Also publish to Redis for cross-instance sync
	r.pubsub.Publish(redis.GetRoomChannel(r.ID.String()), &redis.Message{
		Type:    "yjs-sync",
		From:    r.instanceID,
		Payload: data,
	})
}

// checkIdle checks if the room is idle and should save/cleanup
func (r *Room) checkIdle() {
	r.mu.RLock()
	clientCount := len(r.clients)
	lastActivity := r.lastActivity
	r.mu.RUnlock()

	// Save snapshot if idle for 30 seconds
	if time.Since(lastActivity) > 30*time.Second && r.Doc.GetVersion() > 0 {
		go r.saveSnapshot()
	}

	// If no clients for 5 minutes, room can be cleaned up
	if clientCount == 0 && time.Since(lastActivity) > 5*time.Minute {
		r.cancel()
	}
}

// saveSnapshot saves the current document state as a snapshot
func (r *Room) saveSnapshot() {
	snapshot := r.Doc.GetSnapshot()
	if len(snapshot) == 0 {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := r.db.SaveSnapshot(ctx, r.ID, snapshot)
	if err != nil {
		log.Printf("Failed to save snapshot for room %s: %v", r.ID, err)
	} else {
		log.Printf("Saved snapshot for room %s (version %d)", r.ID, r.Doc.GetVersion())
	}
}

// cleanup cleans up room resources
func (r *Room) cleanup() {
	// Save final snapshot
	r.saveSnapshot()

	// Close all client connections
	r.mu.Lock()
	for _, client := range r.clients {
		client.Close()
	}
	r.clients = nil
	r.mu.Unlock()

	// Unsubscribe from Redis
	r.pubsub.Unsubscribe(redis.GetRoomChannel(r.ID.String()))
	r.pubsub.Unsubscribe(redis.GetPresenceChannel(r.ID.String()))

	log.Printf("Room %s cleaned up", r.ID)
}

// Register registers a client with the room
func (r *Room) Register(client *Client) {
	r.register <- client
}

// Unregister unregisters a client from the room
func (r *Room) Unregister(client *Client) {
	r.unregister <- client
}

// ClientCount returns the number of connected clients
func (r *Room) ClientCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.clients)
}

// LoadSnapshot loads the document from stored snapshot
func (r *Room) LoadSnapshot(ctx context.Context) error {
	snapshot, err := r.db.GetLatestSnapshot(ctx, r.ID)
	if err != nil {
		return err
	}

	if snapshot != nil {
		r.Doc.LoadFromSnapshot(snapshot.Snapshot, uint64(snapshot.Version))
	}

	return nil
}

package collab

import (
	"context"
	"sync"

	"github.com/collab-docs/backend/internal/db"
	"github.com/collab-docs/backend/internal/redis"
	"github.com/google/uuid"
)

// RoomManager manages all active collaboration rooms
type RoomManager struct {
	rooms      map[uuid.UUID]*Room
	mu         sync.RWMutex
	pubsub     *redis.PubSub
	db         *db.DB
	instanceID string
	ctx        context.Context
}

// NewRoomManager creates a new room manager
func NewRoomManager(ctx context.Context, pubsub *redis.PubSub, database *db.DB) *RoomManager {
	return &RoomManager{
		rooms:      make(map[uuid.UUID]*Room),
		pubsub:     pubsub,
		db:         database,
		instanceID: uuid.New().String(),
		ctx:        ctx,
	}
}

// GetOrCreateRoom gets an existing room or creates a new one
func (rm *RoomManager) GetOrCreateRoom(ctx context.Context, docID uuid.UUID) (*Room, error) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	if room, exists := rm.rooms[docID]; exists {
		return room, nil
	}

	// Create new room
	room := NewRoom(rm.ctx, docID, rm.pubsub, rm.db, rm.instanceID)

	// Load existing snapshot
	if err := room.LoadSnapshot(ctx); err != nil {
		return nil, err
	}

	rm.rooms[docID] = room

	// Start room in goroutine
	go rm.runRoom(room)

	return room, nil
}

// runRoom runs a room and cleans up when done
func (rm *RoomManager) runRoom(room *Room) {
	room.Run()

	// Room finished, remove from manager
	rm.mu.Lock()
	delete(rm.rooms, room.ID)
	rm.mu.Unlock()
}

// GetRoom gets an existing room
func (rm *RoomManager) GetRoom(docID uuid.UUID) *Room {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	return rm.rooms[docID]
}

// RoomCount returns the number of active rooms
func (rm *RoomManager) RoomCount() int {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	return len(rm.rooms)
}

// CloseAll closes all rooms
func (rm *RoomManager) CloseAll() {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	for _, room := range rm.rooms {
		room.cancel()
	}
}

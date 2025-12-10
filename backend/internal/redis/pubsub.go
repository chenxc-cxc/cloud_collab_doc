package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"github.com/go-redis/redis/v8"
)

// PubSub handles Redis pub/sub for multi-instance synchronization
type PubSub struct {
	client     *redis.Client
	ctx        context.Context
	cancel     context.CancelFunc
	subs       map[string]*redis.PubSub
	subsMu     sync.RWMutex
	handlers   map[string][]MessageHandler
	handlersMu sync.RWMutex
}

// MessageHandler is a function that handles pub/sub messages
type MessageHandler func(channel string, payload []byte)

// Message represents a pub/sub message
type Message struct {
	Type    string          `json:"type"`
	From    string          `json:"from"`
	Payload json.RawMessage `json:"payload"`
}

// New creates a new PubSub instance
func New(ctx context.Context) (*PubSub, error) {
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		redisURL = "redis://localhost:6379"
	}

	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Redis URL: %w", err)
	}

	client := redis.NewClient(opts)

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	subCtx, cancel := context.WithCancel(ctx)

	return &PubSub{
		client:   client,
		ctx:      subCtx,
		cancel:   cancel,
		subs:     make(map[string]*redis.PubSub),
		handlers: make(map[string][]MessageHandler),
	}, nil
}

// Close closes the PubSub connection
func (ps *PubSub) Close() error {
	ps.cancel()

	ps.subsMu.Lock()
	for _, sub := range ps.subs {
		sub.Close()
	}
	ps.subsMu.Unlock()

	return ps.client.Close()
}

// Subscribe subscribes to a channel
func (ps *PubSub) Subscribe(channel string, handler MessageHandler) error {
	ps.subsMu.Lock()
	defer ps.subsMu.Unlock()

	// Add handler
	ps.handlersMu.Lock()
	ps.handlers[channel] = append(ps.handlers[channel], handler)
	ps.handlersMu.Unlock()

	// Check if already subscribed
	if _, exists := ps.subs[channel]; exists {
		return nil
	}

	// Create subscription
	sub := ps.client.Subscribe(ps.ctx, channel)
	ps.subs[channel] = sub

	// Start listening in goroutine
	go ps.listen(channel, sub)

	return nil
}

// Unsubscribe unsubscribes from a channel
func (ps *PubSub) Unsubscribe(channel string) error {
	ps.subsMu.Lock()
	defer ps.subsMu.Unlock()

	if sub, exists := ps.subs[channel]; exists {
		sub.Close()
		delete(ps.subs, channel)
	}

	ps.handlersMu.Lock()
	delete(ps.handlers, channel)
	ps.handlersMu.Unlock()

	return nil
}

// Publish publishes a message to a channel
func (ps *PubSub) Publish(channel string, msg *Message) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	return ps.client.Publish(ps.ctx, channel, data).Err()
}

// PublishRaw publishes raw bytes to a channel
func (ps *PubSub) PublishRaw(channel string, data []byte) error {
	return ps.client.Publish(ps.ctx, channel, data).Err()
}

// listen listens for messages on a subscription
func (ps *PubSub) listen(channel string, sub *redis.PubSub) {
	ch := sub.Channel()

	for {
		select {
		case <-ps.ctx.Done():
			return
		case msg, ok := <-ch:
			if !ok {
				return
			}

			ps.handlersMu.RLock()
			handlers := ps.handlers[channel]
			ps.handlersMu.RUnlock()

			for _, handler := range handlers {
				go handler(channel, []byte(msg.Payload))
			}
		}
	}
}

// GetRoomChannel returns the channel name for a document room
func GetRoomChannel(docID string) string {
	return fmt.Sprintf("room:%s", docID)
}

// GetPresenceChannel returns the channel name for presence updates
func GetPresenceChannel(docID string) string {
	return fmt.Sprintf("presence:%s", docID)
}

// Set stores a value in Redis
func (ps *PubSub) Set(ctx context.Context, key string, value interface{}) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return ps.client.Set(ctx, key, data, 0).Err()
}

// Get retrieves a value from Redis
func (ps *PubSub) Get(ctx context.Context, key string, dest interface{}) error {
	data, err := ps.client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil
		}
		return err
	}
	return json.Unmarshal(data, dest)
}

// Delete removes a key from Redis
func (ps *PubSub) Delete(ctx context.Context, key string) error {
	return ps.client.Del(ctx, key).Err()
}

// SetBytes stores raw bytes in Redis
func (ps *PubSub) SetBytes(ctx context.Context, key string, value []byte) error {
	return ps.client.Set(ctx, key, value, 0).Err()
}

// GetBytes retrieves raw bytes from Redis
func (ps *PubSub) GetBytes(ctx context.Context, key string) ([]byte, error) {
	data, err := ps.client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, err
	}
	return data, nil
}

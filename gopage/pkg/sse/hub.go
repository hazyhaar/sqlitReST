package sse

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// Message represents an SSE message
type Message struct {
	Event string      `json:"event,omitempty"`
	Data  interface{} `json:"data"`
	ID    string      `json:"id,omitempty"`
	Retry int         `json:"retry,omitempty"`
}

// Client represents an SSE client connection
type Client struct {
	ID       string
	Channel  string
	Messages chan Message
	Done     chan struct{}
}

// Hub manages SSE connections and broadcasts
type Hub struct {
	mu       sync.RWMutex
	clients  map[string]*Client
	channels map[string]map[string]*Client // channel -> clientID -> client
}

// Global hub instance
var globalHub *Hub
var hubOnce sync.Once

// GetHub returns the global SSE hub
func GetHub() *Hub {
	hubOnce.Do(func() {
		globalHub = &Hub{
			clients:  make(map[string]*Client),
			channels: make(map[string]map[string]*Client),
		}
	})
	return globalHub
}

// Register adds a client to the hub
func (h *Hub) Register(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.clients[client.ID] = client

	if client.Channel != "" {
		if h.channels[client.Channel] == nil {
			h.channels[client.Channel] = make(map[string]*Client)
		}
		h.channels[client.Channel][client.ID] = client
	}
}

// Unregister removes a client from the hub
func (h *Hub) Unregister(clientID string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	client, ok := h.clients[clientID]
	if !ok {
		return
	}

	if client.Channel != "" {
		delete(h.channels[client.Channel], clientID)
		if len(h.channels[client.Channel]) == 0 {
			delete(h.channels, client.Channel)
		}
	}

	delete(h.clients, clientID)
	close(client.Done)
}

// Broadcast sends a message to all clients
func (h *Hub) Broadcast(msg Message) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for _, client := range h.clients {
		select {
		case client.Messages <- msg:
		default:
			// Client buffer full, skip
		}
	}
}

// BroadcastToChannel sends a message to all clients in a channel
func (h *Hub) BroadcastToChannel(channel string, msg Message) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	clients, ok := h.channels[channel]
	if !ok {
		return
	}

	for _, client := range clients {
		select {
		case client.Messages <- msg:
		default:
			// Client buffer full, skip
		}
	}
}

// SendToClient sends a message to a specific client
func (h *Hub) SendToClient(clientID string, msg Message) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()

	client, ok := h.clients[clientID]
	if !ok {
		return false
	}

	select {
	case client.Messages <- msg:
		return true
	default:
		return false
	}
}

// ClientCount returns the number of connected clients
func (h *Hub) ClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// ChannelClientCount returns the number of clients in a channel
func (h *Hub) ChannelClientCount(channel string) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.channels[channel])
}

// ListChannels returns all active channels
func (h *Hub) ListChannels() []string {
	h.mu.RLock()
	defer h.mu.RUnlock()

	channels := make([]string, 0, len(h.channels))
	for ch := range h.channels {
		channels = append(channels, ch)
	}
	return channels
}

// Handler returns an HTTP handler for SSE connections
func (h *Hub) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Set SSE headers
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("Access-Control-Allow-Origin", "*")

		// Get channel from query param
		channel := r.URL.Query().Get("channel")

		// Create client
		clientID := fmt.Sprintf("%d", time.Now().UnixNano())
		client := &Client{
			ID:       clientID,
			Channel:  channel,
			Messages: make(chan Message, 100),
			Done:     make(chan struct{}),
		}

		// Register client
		h.Register(client)
		defer h.Unregister(clientID)

		// Get flusher
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "SSE not supported", http.StatusInternalServerError)
			return
		}

		// Send initial connection message
		fmt.Fprintf(w, "event: connected\ndata: {\"clientId\":\"%s\"}\n\n", clientID)
		flusher.Flush()

		// Keep-alive ticker
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		// Event loop
		for {
			select {
			case <-r.Context().Done():
				return
			case <-client.Done:
				return
			case <-ticker.C:
				// Send keep-alive
				fmt.Fprintf(w, ": keepalive\n\n")
				flusher.Flush()
			case msg := <-client.Messages:
				// Format and send message
				if msg.Event != "" {
					fmt.Fprintf(w, "event: %s\n", msg.Event)
				}
				if msg.ID != "" {
					fmt.Fprintf(w, "id: %s\n", msg.ID)
				}
				if msg.Retry > 0 {
					fmt.Fprintf(w, "retry: %d\n", msg.Retry)
				}

				// Marshal data
				var dataStr string
				switch v := msg.Data.(type) {
				case string:
					dataStr = v
				default:
					jsonData, err := json.Marshal(v)
					if err != nil {
						dataStr = fmt.Sprintf("%v", v)
					} else {
						dataStr = string(jsonData)
					}
				}
				fmt.Fprintf(w, "data: %s\n\n", dataStr)
				flusher.Flush()
			}
		}
	}
}

// Notify sends a simple notification
func Notify(event, data string) {
	GetHub().Broadcast(Message{Event: event, Data: data})
}

// NotifyChannel sends a notification to a specific channel
func NotifyChannel(channel, event, data string) {
	GetHub().BroadcastToChannel(channel, Message{Event: event, Data: data})
}

// NotifyJSON sends a JSON notification
func NotifyJSON(event string, data interface{}) {
	GetHub().Broadcast(Message{Event: event, Data: data})
}

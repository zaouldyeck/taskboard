package websocket

import (
	"log"
	"sync"

	"github.com/nats-io/nats.go"
)

// Hub represents internal state of active clients and what messages to broadcast to them.
type Hub struct {
	// Active connections.
	clients map[*Client]bool

	// Queue for messages to send.
	broadcast chan []byte

	// Queue for new clients.
	Register chan *Client

	// Queue for leaving clients.
	unregister chan *Client

	nats *nats.Conn

	// Protect concurrent access to clients map.
	mu sync.RWMutex
}

func NewHub(nc *nats.Conn) *Hub {
	return &Hub{
		broadcast:  make(chan []byte, 256),
		Register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[*Client]bool),
		nats:       nc,
	}
}

// Run starts the hub's main loop.
func (h *Hub) Run() {
	h.nats.Subscribe("tasks.>", func(msg *nats.Msg) {
		log.Printf("📨 Received NATS event on %s, broadcasting to %d clients", msg.Subject, len(h.clients))
		h.broadcast <- msg.Data
	})

	log.Println("✅ Hub subscribed to tasks.* events")

	for {
		select {
		case client := <-h.Register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			log.Printf("➕ Client connected. Total clients: %d", len(h.clients))

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.Send)
			}
			h.mu.Unlock()
			log.Printf("➖ Client disconnected. Total clients: %d", len(h.clients))

		case message := <-h.broadcast:
			h.mu.RLock()
			for client := range h.clients {
				select {
				case client.Send <- message:
					// Message sent successfully.
				default:
					// Clients send buffer full. Closing.
					close(client.Send)
					delete(h.clients, client)
				}
			}
			h.mu.RUnlock()
		}
	}
}

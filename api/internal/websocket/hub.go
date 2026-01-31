package websocket

import (
	"encoding/json"
	"log"
	"sync"

	"github.com/fasthttp/websocket"
	"github.com/google/uuid"
)

// MessageType represents the type of WebSocket message
type MessageType string

const (
	MessageTypeEvent         MessageType = "event"
	MessageTypeAlert         MessageType = "alert"
	MessageTypeServerStatus  MessageType = "server_status"
	MessageTypeServerCreated MessageType = "server_created"
	MessageTypeServerDeleted MessageType = "server_deleted"
	MessageTypeDashboard     MessageType = "dashboard"
)

// Message is sent over WebSocket connections
type Message struct {
	Type    MessageType `json:"type"`
	Payload interface{} `json:"payload"`
}

// Client represents a connected WebSocket client
type Client struct {
	ID             string
	OrganizationID uuid.UUID
	Conn           *websocket.Conn
	Send           chan []byte
	Hub            *Hub
	mu             sync.Mutex
}

// Hub maintains active WebSocket connections and broadcasts messages
type Hub struct {
	// Registered clients by organization
	clients map[uuid.UUID]map[*Client]bool

	// Register requests from clients
	register chan *Client

	// Unregister requests from clients
	unregister chan *Client

	// Broadcast messages to an organization
	broadcast chan *orgMessage

	mu sync.RWMutex
}

type orgMessage struct {
	OrganizationID uuid.UUID
	Data           []byte
}

// NewHub creates a new WebSocket hub
func NewHub() *Hub {
	return &Hub{
		clients:    make(map[uuid.UUID]map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan *orgMessage),
	}
}

// Run starts the hub's main loop
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			if h.clients[client.OrganizationID] == nil {
				h.clients[client.OrganizationID] = make(map[*Client]bool)
			}
			h.clients[client.OrganizationID][client] = true
			h.mu.Unlock()
			log.Printf("WebSocket client %s connected (org: %s)", client.ID, client.OrganizationID)

		case client := <-h.unregister:
			h.mu.Lock()
			if clients, ok := h.clients[client.OrganizationID]; ok {
				if _, exists := clients[client]; exists {
					delete(clients, client)
					close(client.Send)
					if len(clients) == 0 {
						delete(h.clients, client.OrganizationID)
					}
				}
			}
			h.mu.Unlock()
			log.Printf("WebSocket client %s disconnected (org: %s)", client.ID, client.OrganizationID)

		case msg := <-h.broadcast:
			h.mu.RLock()
			if clients, ok := h.clients[msg.OrganizationID]; ok {
				for client := range clients {
					select {
					case client.Send <- msg.Data:
					default:
						// Client send buffer is full, close connection
						close(client.Send)
						delete(clients, client)
					}
				}
			}
			h.mu.RUnlock()
		}
	}
}

// Broadcast sends a message to all clients in an organization
func (h *Hub) Broadcast(orgID uuid.UUID, msgType MessageType, payload interface{}) {
	msg := Message{
		Type:    msgType,
		Payload: payload,
	}
	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("Failed to marshal WebSocket message: %v", err)
		return
	}

	h.broadcast <- &orgMessage{
		OrganizationID: orgID,
		Data:           data,
	}
}

// GetClientCount returns the number of connected clients for an organization
func (h *Hub) GetClientCount(orgID uuid.UUID) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if clients, ok := h.clients[orgID]; ok {
		return len(clients)
	}
	return 0
}

// Write sends a message to the client
func (c *Client) Write(data []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := c.Conn.WriteMessage(websocket.TextMessage, data); err != nil {
		log.Printf("WebSocket write error: %v", err)
	}
}

// WritePump pumps messages from the hub to the WebSocket connection
func (c *Client) WritePump() {
	defer func() {
		c.Conn.Close()
	}()

	for message := range c.Send {
		c.mu.Lock()
		err := c.Conn.WriteMessage(websocket.TextMessage, message)
		c.mu.Unlock()
		if err != nil {
			return
		}
	}
}

// ReadPump pumps messages from the WebSocket connection to the hub
func (c *Client) ReadPump() {
	defer func() {
		c.Hub.unregister <- c
		c.Conn.Close()
	}()

	for {
		_, _, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}
		// We don't need to handle incoming messages for now,
		// but the read pump keeps the connection alive
	}
}

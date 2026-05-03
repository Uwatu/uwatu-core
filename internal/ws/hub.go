package ws

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/gofiber/websocket/v2"
	"github.com/uwatu/uwatu-core/internal/config"
	"github.com/uwatu/uwatu-core/internal/models"
)

type Client struct {
	Conn   *websocket.Conn
	FarmID string
	Role   string
}

type Hub struct {
	mu      sync.RWMutex
	clients map[*Client]bool
}

func NewHub() *Hub {
	return &Hub{clients: make(map[*Client]bool)}
}

func (h *Hub) Register(c *Client) {
	h.mu.Lock()
	h.clients[c] = true
	h.mu.Unlock()
	config.LogInfo("WS", fmt.Sprintf("Client connected: farm=%s role=%s", c.FarmID, c.Role))
}

func (h *Hub) Unregister(c *Client) {
	h.mu.Lock()
	delete(h.clients, c)
	h.mu.Unlock()
	c.Conn.Close()
	config.LogInfo("WS", fmt.Sprintf("Client disconnected: farm=%s role=%s", c.FarmID, c.Role))
}

func (h *Hub) BroadcastEnriched(matrix models.SignalMatrix) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for client := range h.clients {
		data, err := json.Marshal(matrix)
		if err != nil {
			config.LogError("WS", fmt.Sprintf("marshal: %v", err))
			continue
		}
		if err := client.Conn.WriteMessage(websocket.TextMessage, data); err != nil {
			config.LogError("WS", fmt.Sprintf("write: %v", err))
		}
	}
}

func (h *Hub) Upgrade(c *websocket.Conn) {
	farmID := c.Params("farm_id", "default")
	role := c.Query("role", "farmer")
	client := &Client{Conn: c, FarmID: farmID, Role: role}
	h.Register(client)
	defer h.Unregister(client)
	for {
		if _, _, err := c.ReadMessage(); err != nil {
			break
		}
	}
}

package websocket

import (
	"sync"

	"github.com/gofiber/websocket/v2"
)

type Client struct {
	ID   string
	Conn *websocket.Conn
	mu   sync.Mutex
}

// SafeWriteMessage mencegah concurrent write ke koneksi WebSocket
func (c *Client) SafeWriteMessage(messageType int, data []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.Conn.WriteMessage(messageType, data)
}

type Manager struct {
	clients map[*websocket.Conn]*Client
	lock    sync.RWMutex
}

var manager = &Manager{
	clients: make(map[*websocket.Conn]*Client),
}

func GetManager() *Manager {
	return manager
}

func (m *Manager) AddClient(conn *websocket.Conn, client *Client) {
	m.lock.Lock()
	defer m.lock.Unlock()
	m.clients[conn] = client
}

func (m *Manager) RemoveClient(conn *websocket.Conn) {
	m.lock.Lock()
	defer m.lock.Unlock()
	delete(m.clients, conn)
}

func (m *Manager) Broadcast(message []byte) {
	m.lock.RLock()
	defer m.lock.RUnlock()

	for conn, client := range m.clients {
		if err := client.SafeWriteMessage(websocket.TextMessage, message); err != nil {
			conn.Close()
			// aman untuk dihapus di luar lock saat ini karena loop hanya membaca
			go m.RemoveClient(conn)
		}
	}
}

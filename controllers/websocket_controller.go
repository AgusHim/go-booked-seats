package controllers

import (
	"encoding/json"
	"fmt"
	"go-ticketing/models"
	ws "go-ticketing/websocket"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
	"github.com/google/uuid"
)

type WebsocketController struct{}

func NewWebsocketController() WebsocketController {
	return WebsocketController{}
}

// Middleware for upgrading to WebSocket with context-safe extraction
func (c *WebsocketController) UpgradeConnection(ctx *fiber.Ctx) error {
	// Simpan user_id dari query sebelum upgrade
	userID := ctx.Query("user_id", uuid.NewString())

	return websocket.New(func(conn *websocket.Conn) {
		// Atur batasan dan timeout
		conn.SetReadLimit(512)
		conn.SetReadDeadline(time.Now().Add(30 * time.Second))
		conn.SetPongHandler(func(string) error {
			conn.SetReadDeadline(time.Now().Add(30 * time.Second))
			return nil
		})

		client := &ws.Client{
			ID:   userID,
			Conn: conn,
		}

		manager := ws.GetManager()
		manager.AddClient(conn, client)
		defer manager.RemoveClient(conn)

		// Kirim ping setiap 10 detik
		go func() {
			ticker := time.NewTicker(10 * time.Second)
			defer ticker.Stop()

			for range ticker.C {
				if err := client.SafeWriteMessage(websocket.PingMessage, []byte{}); err != nil {
					client.Conn.Close()
					break
				}
			}
		}()

		// Baca pesan dummy untuk menjaga koneksi tetap hidup
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				break
			}
		}
	})(ctx)
}

func (c *WebsocketController) SendWebsocketMessage(message models.Message) error {
	bytes, err := json.Marshal(message)
	if err != nil {
		return err
	}
	manager := ws.GetManager()
	manager.Broadcast(bytes)
	return nil
}

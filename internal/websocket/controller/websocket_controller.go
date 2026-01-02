package controller

import (
	ws "github.com/LautaroBlasco23/impostor/internal/websocket"
	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
)

type WebSocketController struct {
	hub *ws.Hub
}

func NewWebSocketController(hub *ws.Hub) *WebSocketController {
	return &WebSocketController{hub: hub}
}

func (c *WebSocketController) UpgradeMiddleware(ctx *fiber.Ctx) error {
	if websocket.IsWebSocketUpgrade(ctx) {
		return ctx.Next()
	}
	return fiber.ErrUpgradeRequired
}

func (c *WebSocketController) HandleConnection(conn *websocket.Conn) {
	userID := conn.Params("userId")
	roomID := conn.Query("roomId")
	nickname := conn.Query("nickname")

	client := &ws.Client{
		ID:     userID,
		RoomID: roomID,
		Conn:   conn,
		Send:   make(chan []byte, 256),
	}

	c.hub.Register(client)

	if roomID != "" {
		c.hub.BroadcastToRoomExcept(roomID, userID, ws.EventUserJoined, map[string]string{
			"user_id":  userID,
			"nickname": nickname,
		})
	}

	go client.WritePump()
	client.ReadPump(c.hub)
}

func (c *WebSocketController) GetHub() *ws.Hub {
	return c.hub
}

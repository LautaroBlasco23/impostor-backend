package websocket

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"

	"github.com/gofiber/contrib/websocket"
)

type EventType string

const (
	EventUserJoined       EventType = "user_joined"
	EventUserLeft         EventType = "user_left"
	EventUserReady        EventType = "user_ready"
	EventCategorySet      EventType = "category_set"
	EventGameStarted      EventType = "game_started"
	EventUserVoted        EventType = "user_voted"
	EventUserEliminated   EventType = "user_eliminated"
	EventGameWon          EventType = "game_won"
	EventGameLost         EventType = "game_lost"
	EventRoomUpdate       EventType = "room_update"
	EventUserDisconnected EventType = "user_disconnected"
	EventUserReconnected  EventType = "user_reconnected"
	EventGameCancelled    EventType = "game_cancelled"
)

type MessageType string

const (
	MessageTypeReconnect MessageType = "reconnect"
)

type ClientMessage struct {
	Type    MessageType     `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

type ReconnectPayload struct {
	GameID string `json:"game_id"`
}

type Event struct {
	Type    EventType   `json:"type"`
	RoomID  string      `json:"room_id"`
	Payload interface{} `json:"payload"`
}

type Client struct {
	ID        string
	RoomID    string
	Conn      *websocket.Conn
	Send      chan []byte
	hub       *Hub
	closeOnce sync.Once
}

type DisconnectHandler func(clientID, roomID string)

type ReconnectHandler func(clientID, roomID, gameID string)

type Hub struct {
	clients           map[string]*Client
	rooms             map[string]map[string]*Client
	broadcast         chan Event
	register          chan *Client
	unregister        chan *Client
	mu                sync.RWMutex
	disconnectHandler DisconnectHandler
	reconnectHandler  ReconnectHandler
}

func NewHub() *Hub {
	return &Hub{
		clients:    make(map[string]*Client),
		rooms:      make(map[string]map[string]*Client),
		broadcast:  make(chan Event, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

func (h *Hub) SetDisconnectHandler(handler DisconnectHandler) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.disconnectHandler = handler
}

func (h *Hub) SetReconnectHandler(handler ReconnectHandler) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.reconnectHandler = handler
}

func NewClient(id, roomID string, conn *websocket.Conn, hub *Hub) *Client {
	return &Client{
		ID:     id,
		RoomID: roomID,
		Conn:   conn,
		Send:   make(chan []byte, 256),
		hub:    hub,
	}
}

func (c *Client) Close() {
	c.closeOnce.Do(func() {
		if c.Conn != nil {
			err := c.Conn.Close()
			if err != nil {
				fmt.Printf("error closing clients connection!, %v", err)
				fmt.Println()
			}
		}
	})
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.registerClient(client)

		case client := <-h.unregister:
			h.unregisterClient(client)

		case event := <-h.broadcast:
			h.broadcastToRoom(event)
		}
	}
}

func (h *Hub) Register(client *Client) {
	h.register <- client
}

func (h *Hub) Unregister(client *Client) {
	h.unregister <- client
}

func (h *Hub) registerClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if existing, ok := h.clients[client.ID]; ok {
		h.removeClientLocked(existing, false)
	}

	h.clients[client.ID] = client

	if client.RoomID != "" {
		if h.rooms[client.RoomID] == nil {
			h.rooms[client.RoomID] = make(map[string]*Client)
		}
		h.rooms[client.RoomID][client.ID] = client
	}

	log.Printf("Client %s registered to room %s", client.ID, client.RoomID)
}

func (h *Hub) removeClientLocked(client *Client, triggerDisconnect bool) {
	roomID := client.RoomID
	clientID := client.ID

	delete(h.clients, client.ID)

	select {
	case <-client.Send:
	default:
		close(client.Send)
	}

	if client.RoomID != "" {
		if room, exists := h.rooms[client.RoomID]; exists {
			delete(room, client.ID)
			if len(room) == 0 {
				delete(h.rooms, client.RoomID)
			}
		}
	}

	client.Close()

	log.Printf("Client %s unregistered from room %s", clientID, roomID)

	if triggerDisconnect && roomID != "" && h.disconnectHandler != nil {
		go h.disconnectHandler(clientID, roomID)
	}
}

func (h *Hub) unregisterClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, ok := h.clients[client.ID]; ok {
		h.removeClientLocked(client, true)
	}
}

func (h *Hub) broadcastToRoom(event Event) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	room, exists := h.rooms[event.RoomID]
	if !exists {
		return
	}

	message, err := json.Marshal(event)
	if err != nil {
		log.Printf("Error marshaling event: %v", err)
		return
	}

	for _, client := range room {
		select {
		case client.Send <- message:
		default:
			log.Printf("Client %s send buffer full, skipping message", client.ID)
		}
	}
}

func (h *Hub) BroadcastToRoom(roomID string, eventType EventType, payload interface{}) {
	event := Event{
		Type:    eventType,
		RoomID:  roomID,
		Payload: payload,
	}
	h.broadcast <- event
}

func (h *Hub) UpdateClientRoom(clientID, newRoomID string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	client, exists := h.clients[clientID]
	if !exists {
		return
	}

	if client.RoomID != "" {
		if room, exists := h.rooms[client.RoomID]; exists {
			delete(room, clientID)
			if len(room) == 0 {
				delete(h.rooms, client.RoomID)
			}
		}
	}

	client.RoomID = newRoomID

	if newRoomID != "" {
		if h.rooms[newRoomID] == nil {
			h.rooms[newRoomID] = make(map[string]*Client)
		}
		h.rooms[newRoomID][clientID] = client
	}
}

func (c *Client) ReadPump() {
	defer func() {
		c.hub.Unregister(c)
	}()

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			break
		}

		c.handleMessage(message)
	}
}

func (c *Client) handleMessage(message []byte) {
	var msg ClientMessage
	if err := json.Unmarshal(message, &msg); err != nil {
		log.Printf("Error unmarshaling client message: %v", err)
		return
	}

	if msg.Type == MessageTypeReconnect {
		var payload ReconnectPayload
		if err := json.Unmarshal(msg.Payload, &payload); err != nil {
			log.Printf("Error unmarshaling reconnect payload: %v", err)
			return
		}
		c.hub.handleReconnect(c.ID, c.RoomID, payload.GameID)
	}
}

func (h *Hub) handleReconnect(clientID, roomID, gameID string) {
	h.mu.RLock()
	handler := h.reconnectHandler
	h.mu.RUnlock()

	if handler != nil {
		go handler(clientID, roomID, gameID)
	}
}

func (c *Client) WritePump() {
	for message := range c.Send {
		if err := c.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
			break
		}
	}
}

func (h *Hub) BroadcastToRoomExcept(roomID string, excludeClientID string, eventType EventType, payload interface{}) {
	event := Event{
		Type:    eventType,
		RoomID:  roomID,
		Payload: payload,
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	room, exists := h.rooms[roomID]
	if !exists {
		return
	}

	message, err := json.Marshal(event)
	if err != nil {
		log.Printf("Error marshaling event: %v", err)
		return
	}

	for _, client := range room {
		if client.ID == excludeClientID {
			continue
		}
		select {
		case client.Send <- message:
		default:
			log.Printf("Client %s send buffer full, skipping message", client.ID)
		}
	}
}

func (h *Hub) SendToClient(clientID, roomID string, eventType EventType, payload interface{}) {
	event := Event{
		Type:    eventType,
		RoomID:  roomID,
		Payload: payload,
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	client, exists := h.clients[clientID]
	if !exists {
		return
	}

	message, err := json.Marshal(event)
	if err != nil {
		log.Printf("Error marshaling event: %v", err)
		return
	}

	select {
	case client.Send <- message:
	default:
		log.Printf("Client %s send buffer full, skipping message", client.ID)
	}
}

func (h *Hub) IsClientConnected(clientID string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	_, exists := h.clients[clientID]
	return exists
}

func (h *Hub) GetRoomClientIDs(roomID string) []string {
	h.mu.RLock()
	defer h.mu.RUnlock()

	room, exists := h.rooms[roomID]
	if !exists {
		return nil
	}

	ids := make([]string, 0, len(room))
	for id := range room {
		ids = append(ids, id)
	}
	return ids
}

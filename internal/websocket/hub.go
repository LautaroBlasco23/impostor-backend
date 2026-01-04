package websocket

import (
	"encoding/json"
	"log"
	"sync"

	"github.com/gofiber/contrib/websocket"
)

type EventType string

const (
	EventUserJoined     EventType = "user_joined"
	EventUserLeft       EventType = "user_left"
	EventUserReady      EventType = "user_ready"
	EventCategorySet    EventType = "category_set"
	EventGameStarted    EventType = "game_started"
	EventUserVoted      EventType = "user_voted"
	EventUserEliminated EventType = "user_eliminated"
	EventGameWon        EventType = "game_won"
	EventGameLost       EventType = "game_lost"
	EventRoomUpdate     EventType = "room_update"
)

type Event struct {
	Type    EventType   `json:"type"`
	RoomID  string      `json:"room_id"`
	Payload interface{} `json:"payload"`
}

type Client struct {
	ID     string
	RoomID string
	Conn   *websocket.Conn
	Send   chan []byte
}

type Hub struct {
	clients    map[string]*Client
	rooms      map[string]map[string]*Client
	broadcast  chan Event
	register   chan *Client
	unregister chan *Client
	mu         sync.RWMutex
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

	h.clients[client.ID] = client

	if client.RoomID != "" {
		if h.rooms[client.RoomID] == nil {
			h.rooms[client.RoomID] = make(map[string]*Client)
		}
		h.rooms[client.RoomID][client.ID] = client
	}

	log.Printf("Client %s registered to room %s", client.ID, client.RoomID)
}

func (h *Hub) unregisterClient(client *Client) {
	h.mu.Lock()

	if _, ok := h.clients[client.ID]; ok {
		roomID := client.RoomID
		clientID := client.ID

		delete(h.clients, client.ID)
		close(client.Send)

		if client.RoomID != "" {
			if room, exists := h.rooms[client.RoomID]; exists {
				delete(room, client.ID)
				if len(room) == 0 {
					delete(h.rooms, client.RoomID)
				}
			}
		}

		log.Printf("Client %s unregistered from room %s", client.ID, client.RoomID)

		h.mu.Unlock()

		if roomID != "" {
			h.BroadcastToRoom(roomID, EventUserLeft, map[string]string{
				"user_id": clientID,
			})
		}
		return
	}

	h.mu.Unlock()
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
			close(client.Send)
			delete(h.clients, client.ID)
			delete(room, client.ID)
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

func (c *Client) ReadPump(hub *Hub) {
	defer func() {
		hub.Unregister(c)
		err := c.Conn.Close()
		if err != nil {
			log.Fatalf("error closing client connection: %v", err)
		}
	}()

	for {
		_, _, err := c.Conn.ReadMessage()
		if err != nil {
			break
		}
	}
}

func (c *Client) WritePump() {
	defer func() {
		err := c.Conn.Close()
		if err != nil {
			log.Fatalf("error closing client connection, %v ", err)
		}
	}()

	for message := range c.Send {
		err := c.Conn.WriteMessage(websocket.TextMessage, message)
		if err != nil {
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
			close(client.Send)
			delete(h.clients, client.ID)
			delete(room, client.ID)
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
		close(client.Send)
		delete(h.clients, client.ID)
	}
}

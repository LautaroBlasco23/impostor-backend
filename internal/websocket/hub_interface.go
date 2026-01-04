package websocket

type HubBroadcaster interface {
	BroadcastToRoom(roomID string, eventType EventType, payload interface{})
	BroadcastToRoomExcept(roomID string, excludeClientID string, eventType EventType, payload interface{})
	UpdateClientRoom(clientID, newRoomID string)
	SendToClient(clientID, roomID string, eventType EventType, payload interface{})
}

package service_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/LautaroBlasco23/impostor/internal/core/user/model"
	"github.com/LautaroBlasco23/impostor/internal/core/user/service"
	ws "github.com/LautaroBlasco23/impostor/internal/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func createTestClient(id, roomID string, eventCh chan<- ws.Event) *ws.Client {
	client := &ws.Client{
		ID:     id,
		RoomID: roomID,
		Conn:   nil,
		Send:   make(chan []byte, 256),
	}

	go func() {
		for msg := range client.Send {
			var event ws.Event
			if err := json.Unmarshal(msg, &event); err == nil {
				select {
				case eventCh <- event:
				default:
				}
			}
		}
	}()

	return client
}

func TestJoinRoom_BroadcastsEvent_Integration(t *testing.T) {
	repo := new(MockUserRepository)
	hub := ws.NewHub()

	go hub.Run()
	time.Sleep(10 * time.Millisecond)

	eventReceived := make(chan ws.Event, 1)
	existingClient := createTestClient("existing-user", "room-456", eventReceived)
	hub.Register(existingClient)
	time.Sleep(10 * time.Millisecond)

	user := &model.User{ID: "user-123", Nickname: "newPlayer", RoomID: ""}
	repo.On("GetByID", mock.Anything, "user-123").Return(user, nil)
	repo.On("Update", mock.Anything, mock.AnythingOfType("*model.User")).Return(nil)

	svc := service.NewUserService(repo, hub)
	err := svc.JoinRoom(context.Background(), "user-123", "room-456")

	assert.NoError(t, err)

	select {
	case event := <-eventReceived:
		assert.Equal(t, ws.EventUserJoined, event.Type)
		assert.Equal(t, "room-456", event.RoomID)
		payload := event.Payload.(map[string]interface{})
		assert.Equal(t, "user-123", payload["user_id"])
		assert.Equal(t, "newPlayer", payload["nickname"])
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timeout waiting for broadcast event")
	}
}

func TestToggleReady_BroadcastsToRoom_Integration(t *testing.T) {
	repo := new(MockUserRepository)
	hub := ws.NewHub()

	go hub.Run()
	time.Sleep(10 * time.Millisecond)

	eventReceived := make(chan ws.Event, 1)
	existingClient := createTestClient("other-user", "room-1", eventReceived)
	hub.Register(existingClient)
	time.Sleep(10 * time.Millisecond)

	user := &model.User{ID: "user-123", Nickname: "player", RoomID: "room-1", IsReady: false}
	repo.On("GetByID", mock.Anything, "user-123").Return(user, nil)
	repo.On("Update", mock.Anything, mock.AnythingOfType("*model.User")).Return(nil)

	svc := service.NewUserService(repo, hub)
	err := svc.ToggleReady(context.Background(), "user-123")

	assert.NoError(t, err)

	select {
	case event := <-eventReceived:
		assert.Equal(t, ws.EventUserReady, event.Type)
		payload := event.Payload.(map[string]interface{})
		assert.Equal(t, true, payload["is_ready"])
		assert.Equal(t, "user-123", payload["user_id"])
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timeout waiting for broadcast event")
	}
}

func TestDeleteUser_BroadcastsUserLeft_Integration(t *testing.T) {
	repo := new(MockUserRepository)
	hub := ws.NewHub()

	go hub.Run()
	time.Sleep(10 * time.Millisecond)

	eventReceived := make(chan ws.Event, 1)
	remainingClient := createTestClient("remaining-user", "room-1", eventReceived)
	hub.Register(remainingClient)
	time.Sleep(10 * time.Millisecond)

	user := &model.User{ID: "leaving-user", Nickname: "leaver", RoomID: "room-1"}
	repo.On("GetByID", mock.Anything, "leaving-user").Return(user, nil)
	repo.On("Delete", mock.Anything, "leaving-user").Return(nil)

	svc := service.NewUserService(repo, hub)
	err := svc.DeleteUser(context.Background(), "leaving-user")

	assert.NoError(t, err)

	select {
	case event := <-eventReceived:
		assert.Equal(t, ws.EventUserLeft, event.Type)
		payload := event.Payload.(map[string]interface{})
		assert.Equal(t, "leaving-user", payload["user_id"])
		assert.Equal(t, "leaver", payload["nickname"])
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timeout waiting for broadcast event")
	}
}

func TestMultipleUsersReceiveBroadcast_Integration(t *testing.T) {
	repo := new(MockUserRepository)
	hub := ws.NewHub()

	go hub.Run()
	time.Sleep(10 * time.Millisecond)

	event1 := make(chan ws.Event, 1)
	event2 := make(chan ws.Event, 1)
	event3 := make(chan ws.Event, 1)

	client1 := createTestClient("client-1", "room-1", event1)
	client2 := createTestClient("client-2", "room-1", event2)
	client3 := createTestClient("client-3", "room-1", event3)

	hub.Register(client1)
	hub.Register(client2)
	hub.Register(client3)
	time.Sleep(10 * time.Millisecond)

	user := &model.User{ID: "new-user", Nickname: "newbie", RoomID: ""}
	repo.On("GetByID", mock.Anything, "new-user").Return(user, nil)
	repo.On("Update", mock.Anything, mock.AnythingOfType("*model.User")).Return(nil)

	svc := service.NewUserService(repo, hub)
	err := svc.JoinRoom(context.Background(), "new-user", "room-1")

	assert.NoError(t, err)

	timeout := time.After(100 * time.Millisecond)
	receivedCount := 0

	for receivedCount < 3 {
		select {
		case e := <-event1:
			assert.Equal(t, ws.EventUserJoined, e.Type)
			receivedCount++
		case e := <-event2:
			assert.Equal(t, ws.EventUserJoined, e.Type)
			receivedCount++
		case e := <-event3:
			assert.Equal(t, ws.EventUserJoined, e.Type)
			receivedCount++
		case <-timeout:
			t.Fatalf("timeout: only received %d/3 events", receivedCount)
		}
	}
}

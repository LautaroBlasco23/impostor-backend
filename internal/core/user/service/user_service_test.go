package service_test

import (
	"context"
	"errors"
	"testing"

	"github.com/LautaroBlasco23/impostor/internal/core/user/model"
	"github.com/LautaroBlasco23/impostor/internal/core/user/service"
	ws "github.com/LautaroBlasco23/impostor/internal/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockUserRepository struct {
	mock.Mock
}

func (m *MockUserRepository) Create(ctx context.Context, user *model.User) error {
	return m.Called(ctx, user).Error(0)
}

func (m *MockUserRepository) GetByID(ctx context.Context, id string) (*model.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.User), args.Error(1)
}

func (m *MockUserRepository) GetByRoomID(ctx context.Context, roomID string) ([]*model.User, error) {
	args := m.Called(ctx, roomID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.User), args.Error(1)
}

func (m *MockUserRepository) Update(ctx context.Context, user *model.User) error {
	return m.Called(ctx, user).Error(0)
}

func (m *MockUserRepository) Delete(ctx context.Context, id string) error {
	return m.Called(ctx, id).Error(0)
}

type MockHubBroadcaster struct {
	mock.Mock
	Events []BroadcastCall
}

type BroadcastCall struct {
	RoomID    string
	EventType ws.EventType
	Payload   interface{}
}

func NewMockHub() *MockHubBroadcaster {
	return &MockHubBroadcaster{Events: make([]BroadcastCall, 0)}
}

func (m *MockHubBroadcaster) BroadcastToRoom(roomID string, eventType ws.EventType, payload interface{}) {
	m.Called(roomID, eventType, payload)
	m.Events = append(m.Events, BroadcastCall{roomID, eventType, payload})
}

func (m *MockHubBroadcaster) BroadcastToRoomExcept(roomID, excludeClientID string, eventType ws.EventType, payload interface{}) {
	m.Called(roomID, excludeClientID, eventType, payload)
}

func (m *MockHubBroadcaster) UpdateClientRoom(clientID, newRoomID string) {
	m.Called(clientID, newRoomID)
}

func (m *MockHubBroadcaster) SendToClient(clientID, roomID string, eventType ws.EventType, payload interface{}) {
	m.Called(clientID, roomID, eventType, payload)
}

func TestCreateUser(t *testing.T) {
	tests := []struct {
		name      string
		req       *model.CreateUserRequest
		repoErr   error
		wantErr   bool
		checkUser func(*testing.T, *model.User)
	}{
		{
			name:    "successful creation",
			req:     &model.CreateUserRequest{Nickname: "player1"},
			repoErr: nil,
			wantErr: false,
			checkUser: func(t *testing.T, u *model.User) {
				assert.NotEmpty(t, u.ID)
				assert.Equal(t, "player1", u.Nickname)
				assert.Equal(t, model.RolePlayer, u.Role)
				assert.False(t, u.IsReady)
				assert.True(t, u.IsAlive)
			},
		},
		{
			name:    "repository error",
			req:     &model.CreateUserRequest{Nickname: "player1"},
			repoErr: errors.New("db error"),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := new(MockUserRepository)
			hub := NewMockHub()

			repo.On("Create", mock.Anything, mock.AnythingOfType("*model.User")).Return(tt.repoErr)

			svc := service.NewUserService(repo, hub)
			user, err := svc.CreateUser(context.Background(), tt.req)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, user)
			} else {
				assert.NoError(t, err)
				tt.checkUser(t, user)
			}
			repo.AssertExpectations(t)
		})
	}
}

func TestGetUser(t *testing.T) {
	tests := []struct {
		name     string
		userID   string
		repoUser *model.User
		repoErr  error
		wantErr  bool
	}{
		{
			name:     "user found",
			userID:   "user-123",
			repoUser: &model.User{ID: "user-123", Nickname: "test"},
			wantErr:  false,
		},
		{
			name:    "user not found",
			userID:  "nonexistent",
			repoErr: errors.New("user not found"),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := new(MockUserRepository)
			hub := NewMockHub()

			repo.On("GetByID", mock.Anything, tt.userID).Return(tt.repoUser, tt.repoErr)

			svc := service.NewUserService(repo, hub)
			user, err := svc.GetUser(context.Background(), tt.userID)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.repoUser, user)
			}
		})
	}
}

func TestGetUsersByRoom(t *testing.T) {
	tests := []struct {
		name    string
		roomID  string
		users   []*model.User
		repoErr error
		wantErr bool
	}{
		{
			name:   "users found",
			roomID: "room-1",
			users: []*model.User{
				{ID: "1", Nickname: "player1"},
				{ID: "2", Nickname: "player2"},
			},
			wantErr: false,
		},
		{
			name:    "repository error",
			roomID:  "room-1",
			repoErr: errors.New("db error"),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := new(MockUserRepository)
			hub := NewMockHub()

			repo.On("GetByRoomID", mock.Anything, tt.roomID).Return(tt.users, tt.repoErr)

			svc := service.NewUserService(repo, hub)
			users, err := svc.GetUsersByRoom(context.Background(), tt.roomID)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.users, users)
			}
		})
	}
}

func TestJoinRoom(t *testing.T) {
	tests := []struct {
		name          string
		userID        string
		roomID        string
		existUser     *model.User
		getErr        error
		updateErr     error
		wantErr       bool
		wantErrMsg    string
		wantBroadcast bool
	}{
		{
			name:          "successful join",
			userID:        "user-123",
			roomID:        "room-456",
			existUser:     &model.User{ID: "user-123", Nickname: "test", RoomID: ""},
			wantBroadcast: true,
		},
		{
			name:       "user already in room",
			userID:     "user-123",
			roomID:     "room-456",
			existUser:  &model.User{ID: "user-123", Nickname: "test", RoomID: "other-room"},
			wantErr:    true,
			wantErrMsg: "user already in a room",
		},
		{
			name:    "user not found",
			userID:  "nonexistent",
			roomID:  "room-456",
			getErr:  errors.New("user not found"),
			wantErr: true,
		},
		{
			name:      "update fails",
			userID:    "user-123",
			roomID:    "room-456",
			existUser: &model.User{ID: "user-123", Nickname: "test", RoomID: ""},
			updateErr: errors.New("update error"),
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := new(MockUserRepository)
			hub := NewMockHub()

			repo.On("GetByID", mock.Anything, tt.userID).Return(tt.existUser, tt.getErr)

			if tt.existUser != nil && tt.existUser.RoomID == "" {
				repo.On("Update", mock.Anything, mock.AnythingOfType("*model.User")).Return(tt.updateErr)
				if tt.updateErr == nil {
					hub.On("UpdateClientRoom", tt.userID, tt.roomID).Return()
					hub.On("BroadcastToRoom", tt.roomID, ws.EventUserJoined, mock.Anything).Return()
				}
			}

			svc := service.NewUserService(repo, hub)
			err := svc.JoinRoom(context.Background(), tt.userID, tt.roomID)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.wantErrMsg != "" {
					assert.Contains(t, err.Error(), tt.wantErrMsg)
				}
			} else {
				assert.NoError(t, err)
			}

			if tt.wantBroadcast {
				assert.Len(t, hub.Events, 1)
				assert.Equal(t, tt.roomID, hub.Events[0].RoomID)
				assert.Equal(t, ws.EventUserJoined, hub.Events[0].EventType)

				payload := hub.Events[0].Payload.(map[string]interface{})
				assert.Equal(t, tt.userID, payload["user_id"])
				assert.Equal(t, tt.existUser.Nickname, payload["nickname"])
			}

			repo.AssertExpectations(t)
		})
	}
}

func TestToggleReady(t *testing.T) {
	tests := []struct {
		name          string
		userID        string
		existUser     *model.User
		getErr        error
		updateErr     error
		wantErr       bool
		expectedReady bool
	}{
		{
			name:          "toggle false to true",
			userID:        "user-123",
			existUser:     &model.User{ID: "user-123", Nickname: "test", RoomID: "room-1", IsReady: false},
			expectedReady: true,
		},
		{
			name:          "toggle true to false",
			userID:        "user-123",
			existUser:     &model.User{ID: "user-123", Nickname: "test", RoomID: "room-1", IsReady: true},
			expectedReady: false,
		},
		{
			name:    "user not found",
			userID:  "nonexistent",
			getErr:  errors.New("user not found"),
			wantErr: true,
		},
		{
			name:          "update fails",
			userID:        "user-123",
			existUser:     &model.User{ID: "user-123", Nickname: "test", RoomID: "room-1", IsReady: false},
			updateErr:     errors.New("update error"),
			wantErr:       true,
			expectedReady: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := new(MockUserRepository)
			hub := NewMockHub()

			repo.On("GetByID", mock.Anything, tt.userID).Return(tt.existUser, tt.getErr)

			if tt.existUser != nil {
				repo.On("Update", mock.Anything, mock.AnythingOfType("*model.User")).Return(tt.updateErr)

				if tt.updateErr == nil {
					hub.On("BroadcastToRoom", tt.existUser.RoomID, ws.EventUserReady, mock.Anything).Return()
				}
			}

			svc := service.NewUserService(repo, hub)
			err := svc.ToggleReady(context.Background(), tt.userID)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Len(t, hub.Events, 1)
				assert.Equal(t, ws.EventUserReady, hub.Events[0].EventType)

				payload := hub.Events[0].Payload.(map[string]interface{})
				assert.Equal(t, tt.expectedReady, payload["is_ready"])
			}
		})
	}
}

func TestCheckAllReady(t *testing.T) {
	tests := []struct {
		name      string
		roomID    string
		users     []*model.User
		repoErr   error
		wantReady bool
		wantErr   bool
	}{
		{
			name:   "all ready with 2 players",
			roomID: "room-1",
			users: []*model.User{
				{ID: "1", IsReady: true},
				{ID: "2", IsReady: true},
			},
			wantReady: true,
		},
		{
			name:   "all ready with 4 players",
			roomID: "room-1",
			users: []*model.User{
				{ID: "1", IsReady: true},
				{ID: "2", IsReady: true},
				{ID: "3", IsReady: true},
				{ID: "4", IsReady: true},
			},
			wantReady: true,
		},
		{
			name:   "one not ready",
			roomID: "room-1",
			users: []*model.User{
				{ID: "1", IsReady: true},
				{ID: "2", IsReady: false},
			},
			wantReady: false,
		},
		{
			name:      "only 1 player",
			roomID:    "room-1",
			users:     []*model.User{{ID: "1", IsReady: true}},
			wantReady: false,
		},
		{
			name:      "empty room",
			roomID:    "room-1",
			users:     []*model.User{},
			wantReady: false,
		},
		{
			name:    "repository error",
			roomID:  "room-1",
			repoErr: errors.New("db error"),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := new(MockUserRepository)
			hub := NewMockHub()

			repo.On("GetByRoomID", mock.Anything, tt.roomID).Return(tt.users, tt.repoErr)

			svc := service.NewUserService(repo, hub)
			ready, err := svc.CheckAllReady(context.Background(), tt.roomID)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantReady, ready)
			}
		})
	}
}

func TestDeleteUser(t *testing.T) {
	tests := []struct {
		name          string
		userID        string
		existUser     *model.User
		getErr        error
		deleteErr     error
		wantErr       bool
		wantBroadcast bool
	}{
		{
			name:          "delete user in room broadcasts event",
			userID:        "user-123",
			existUser:     &model.User{ID: "user-123", Nickname: "test", RoomID: "room-1"},
			wantBroadcast: true,
		},
		{
			name:          "delete user without room no broadcast",
			userID:        "user-123",
			existUser:     &model.User{ID: "user-123", Nickname: "test", RoomID: ""},
			wantBroadcast: false,
		},
		{
			name:    "user not found",
			userID:  "nonexistent",
			getErr:  errors.New("user not found"),
			wantErr: true,
		},
		{
			name:      "delete fails",
			userID:    "user-123",
			existUser: &model.User{ID: "user-123", Nickname: "test", RoomID: "room-1"},
			deleteErr: errors.New("delete error"),
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := new(MockUserRepository)
			hub := NewMockHub()

			repo.On("GetByID", mock.Anything, tt.userID).Return(tt.existUser, tt.getErr)

			if tt.existUser != nil {
				repo.On("Delete", mock.Anything, tt.userID).Return(tt.deleteErr)
				if tt.existUser.RoomID != "" {
					hub.On("BroadcastToRoom", tt.existUser.RoomID, ws.EventUserLeft, mock.Anything).Return()
				}
			}

			svc := service.NewUserService(repo, hub)
			err := svc.DeleteUser(context.Background(), tt.userID)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			if tt.wantBroadcast && !tt.wantErr {
				assert.Len(t, hub.Events, 1)
				assert.Equal(t, ws.EventUserLeft, hub.Events[0].EventType)

				payload := hub.Events[0].Payload.(map[string]interface{})
				assert.Equal(t, tt.userID, payload["user_id"])
			} else if !tt.wantBroadcast && !tt.wantErr {
				assert.Empty(t, hub.Events)
			}
		})
	}
}

package controller_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http/httptest"
	"testing"

	"github.com/LautaroBlasco23/impostor/internal/core/user/controller"
	"github.com/LautaroBlasco23/impostor/internal/core/user/model"
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockUserService struct {
	mock.Mock
}

func (m *MockUserService) CreateUser(ctx context.Context, req *model.CreateUserRequest) (*model.User, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.User), args.Error(1)
}

func (m *MockUserService) GetUser(ctx context.Context, id string) (*model.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.User), args.Error(1)
}

func (m *MockUserService) GetUsersByRoom(ctx context.Context, roomID string) ([]*model.User, error) {
	args := m.Called(ctx, roomID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.User), args.Error(1)
}

func (m *MockUserService) JoinRoom(ctx context.Context, userID, roomID string) error {
	return m.Called(ctx, userID, roomID).Error(0)
}

func (m *MockUserService) ToggleReady(ctx context.Context, userID string) error {
	return m.Called(ctx, userID).Error(0)
}

func (m *MockUserService) CheckAllReady(ctx context.Context, roomID string) (bool, error) {
	args := m.Called(ctx, roomID)
	return args.Bool(0), args.Error(1)
}

func (m *MockUserService) DeleteUser(ctx context.Context, id string) error {
	return m.Called(ctx, id).Error(0)
}

func setupTestApp(svc *MockUserService) *fiber.App {
	app := fiber.New()
	ctrl := controller.NewUserController(svc)

	app.Post("/users", ctrl.CreateUser)
	app.Get("/users/:id", ctrl.GetUser)
	app.Get("/users/room/:roomId", ctrl.GetUsersByRoom)
	app.Post("/users/:id/join", ctrl.JoinRoom)
	app.Post("/users/:id/ready", ctrl.ToggleReady)
	app.Delete("/users/:id", ctrl.DeleteUser)

	return app
}

func TestCreateUserController(t *testing.T) {
	tests := []struct {
		name           string
		body           interface{}
		mockReturn     *model.User
		mockErr        error
		expectedStatus int
		checkResponse  func(*testing.T, []byte)
	}{
		{
			name:           "successful creation",
			body:           model.CreateUserRequest{Nickname: "player1"},
			mockReturn:     &model.User{ID: "123", Nickname: "player1", Role: model.RolePlayer},
			expectedStatus: fiber.StatusCreated,
			checkResponse: func(t *testing.T, body []byte) {
				var user model.User
				err := json.Unmarshal(body, &user)
				assert.NoError(t, err)
				assert.Equal(t, "player1", user.Nickname)
				assert.Equal(t, "123", user.ID)
			},
		},
		{
			name:           "invalid json body",
			body:           "not valid json",
			expectedStatus: fiber.StatusBadRequest,
			checkResponse: func(t *testing.T, body []byte) {
				var resp map[string]string
				json.Unmarshal(body, &resp)
				assert.Contains(t, resp["error"], "Invalid request body")
			},
		},
		{
			name:           "service error",
			body:           model.CreateUserRequest{Nickname: "player1"},
			mockErr:        errors.New("database error"),
			expectedStatus: fiber.StatusInternalServerError,
			checkResponse: func(t *testing.T, body []byte) {
				var resp map[string]string
				json.Unmarshal(body, &resp)
				assert.Equal(t, "database error", resp["error"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := new(MockUserService)
			app := setupTestApp(svc)

			if _, ok := tt.body.(model.CreateUserRequest); ok {
				svc.On("CreateUser", mock.Anything, mock.AnythingOfType("*model.CreateUserRequest")).
					Return(tt.mockReturn, tt.mockErr)
			}

			var bodyReader io.Reader
			if str, ok := tt.body.(string); ok {
				bodyReader = bytes.NewBufferString(str)
			} else {
				bodyBytes, _ := json.Marshal(tt.body)
				bodyReader = bytes.NewBuffer(bodyBytes)
			}

			req := httptest.NewRequest("POST", "/users", bodyReader)
			req.Header.Set("Content-Type", "application/json")

			resp, err := app.Test(req)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			if tt.checkResponse != nil {
				body, _ := io.ReadAll(resp.Body)
				tt.checkResponse(t, body)
			}
		})
	}
}

func TestGetUserController(t *testing.T) {
	tests := []struct {
		name           string
		userID         string
		mockReturn     *model.User
		mockErr        error
		expectedStatus int
	}{
		{
			name:           "user found",
			userID:         "123",
			mockReturn:     &model.User{ID: "123", Nickname: "test"},
			expectedStatus: fiber.StatusOK,
		},
		{
			name:           "user not found",
			userID:         "nonexistent",
			mockErr:        errors.New("user not found"),
			expectedStatus: fiber.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := new(MockUserService)
			app := setupTestApp(svc)

			svc.On("GetUser", mock.Anything, tt.userID).Return(tt.mockReturn, tt.mockErr)

			req := httptest.NewRequest("GET", "/users/"+tt.userID, nil)

			resp, err := app.Test(req)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedStatus, resp.StatusCode)
		})
	}
}

func TestGetUsersByRoomController(t *testing.T) {
	tests := []struct {
		name           string
		roomID         string
		mockReturn     []*model.User
		mockErr        error
		expectedStatus int
		expectedCount  int
	}{
		{
			name:   "users found",
			roomID: "room-1",
			mockReturn: []*model.User{
				{ID: "1", Nickname: "player1"},
				{ID: "2", Nickname: "player2"},
			},
			expectedStatus: fiber.StatusOK,
			expectedCount:  2,
		},
		{
			name:           "empty room",
			roomID:         "room-1",
			mockReturn:     []*model.User{},
			expectedStatus: fiber.StatusOK,
			expectedCount:  0,
		},
		{
			name:           "service error",
			roomID:         "room-1",
			mockErr:        errors.New("db error"),
			expectedStatus: fiber.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := new(MockUserService)
			app := setupTestApp(svc)

			svc.On("GetUsersByRoom", mock.Anything, tt.roomID).Return(tt.mockReturn, tt.mockErr)

			req := httptest.NewRequest("GET", "/users/room/"+tt.roomID, nil)

			resp, err := app.Test(req)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			if tt.mockErr == nil {
				body, _ := io.ReadAll(resp.Body)
				var users []*model.User
				json.Unmarshal(body, &users)
				assert.Len(t, users, tt.expectedCount)
			}
		})
	}
}

func TestJoinRoomController(t *testing.T) {
	tests := []struct {
		name           string
		userID         string
		body           interface{}
		mockErr        error
		expectedStatus int
	}{
		{
			name:           "successful join",
			userID:         "user-123",
			body:           model.JoinRoomRequest{RoomID: "room-456"},
			expectedStatus: fiber.StatusOK,
		},
		{
			name:           "invalid body",
			userID:         "user-123",
			body:           "invalid",
			expectedStatus: fiber.StatusBadRequest,
		},
		{
			name:           "user already in room",
			userID:         "user-123",
			body:           model.JoinRoomRequest{RoomID: "room-456"},
			mockErr:        errors.New("user already in a room"),
			expectedStatus: fiber.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := new(MockUserService)
			app := setupTestApp(svc)

			if _, ok := tt.body.(model.JoinRoomRequest); ok {
				svc.On("JoinRoom", mock.Anything, tt.userID, mock.AnythingOfType("string")).
					Return(tt.mockErr)
			}

			var bodyReader io.Reader
			if str, ok := tt.body.(string); ok {
				bodyReader = bytes.NewBufferString(str)
			} else {
				bodyBytes, _ := json.Marshal(tt.body)
				bodyReader = bytes.NewBuffer(bodyBytes)
			}

			req := httptest.NewRequest("POST", "/users/"+tt.userID+"/join", bodyReader)
			req.Header.Set("Content-Type", "application/json")

			resp, err := app.Test(req)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedStatus, resp.StatusCode)
		})
	}
}

func TestToggleReadyController(t *testing.T) {
	tests := []struct {
		name           string
		userID         string
		mockErr        error
		expectedStatus int
	}{
		{
			name:           "successful toggle",
			userID:         "user-123",
			expectedStatus: fiber.StatusOK,
		},
		{
			name:           "service error",
			userID:         "user-123",
			mockErr:        errors.New("user not found"),
			expectedStatus: fiber.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := new(MockUserService)
			app := setupTestApp(svc)

			svc.On("ToggleReady", mock.Anything, tt.userID).Return(tt.mockErr)

			req := httptest.NewRequest("POST", "/users/"+tt.userID+"/ready", nil)

			resp, err := app.Test(req)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedStatus, resp.StatusCode)
		})
	}
}

func TestDeleteUserController(t *testing.T) {
	tests := []struct {
		name           string
		userID         string
		mockErr        error
		expectedStatus int
	}{
		{
			name:           "successful delete",
			userID:         "user-123",
			expectedStatus: fiber.StatusNoContent,
		},
		{
			name:           "service error",
			userID:         "user-123",
			mockErr:        errors.New("delete failed"),
			expectedStatus: fiber.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := new(MockUserService)
			app := setupTestApp(svc)

			svc.On("DeleteUser", mock.Anything, tt.userID).Return(tt.mockErr)

			req := httptest.NewRequest("DELETE", "/users/"+tt.userID, nil)

			resp, err := app.Test(req)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedStatus, resp.StatusCode)
		})
	}
}

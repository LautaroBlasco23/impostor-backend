package controller

import (
	"github.com/LautaroBlasco23/impostor/internal/core/room/model"
	"github.com/LautaroBlasco23/impostor/internal/core/room/service"
	"github.com/gofiber/fiber/v2"
)

type RoomController struct {
	service service.RoomService
}

func NewRoomController(service service.RoomService) *RoomController {
	return &RoomController{service: service}
}

func (c *RoomController) CreateRoom(ctx *fiber.Ctx) error {
	var req model.CreateRoomRequest
	if err := ctx.BodyParser(&req); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	room, err := c.service.CreateRoom(ctx.Context(), &req)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return ctx.Status(fiber.StatusCreated).JSON(room)
}

func (c *RoomController) GetRoom(ctx *fiber.Ctx) error {
	id := ctx.Params("id")
	room, err := c.service.GetRoom(ctx.Context(), id)
	if err != nil {
		return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return ctx.JSON(room)
}

func (c *RoomController) GetAllRooms(ctx *fiber.Ctx) error {
	rooms, err := c.service.GetAllRooms(ctx.Context())
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return ctx.JSON(rooms)
}

func (c *RoomController) SetCategory(ctx *fiber.Ctx) error {
	roomID := ctx.Params("id")

	var req struct {
		Category string `json:"category" validate:"required"`
		LeaderID string `json:"leader_id" validate:"required"`
	}

	if err := ctx.BodyParser(&req); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if req.Category == "" {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Category is required",
		})
	}

	if req.LeaderID == "" {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Leader ID is required",
		})
	}

	if err := c.service.SetCategory(ctx.Context(), roomID, req.LeaderID, req.Category); err != nil {
		return ctx.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return ctx.JSON(fiber.Map{
		"message": "Category set successfully",
	})
}

func (c *RoomController) KickUser(ctx *fiber.Ctx) error {
	roomID := ctx.Params("id")
	targetUserID := ctx.Params("userId")

	var req struct {
		LeaderID string `json:"leader_id"`
	}
	if err := ctx.BodyParser(&req); err != nil || req.LeaderID == "" {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "leader_id is required",
		})
	}

	if err := c.service.KickUser(ctx.Context(), roomID, req.LeaderID, targetUserID); err != nil {
		return ctx.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return ctx.SendStatus(fiber.StatusNoContent)
}

func (c *RoomController) DeleteRoom(ctx *fiber.Ctx) error {
	id := ctx.Params("id")
	var req struct {
		LeaderID string `json:"leader_id" validate:"required"`
	}
	if err := ctx.BodyParser(&req); err != nil || req.LeaderID == "" {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "leader_id is required",
		})
	}
	if err := c.service.DeleteRoom(ctx.Context(), id, req.LeaderID); err != nil {
		return ctx.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
	return ctx.SendStatus(fiber.StatusNoContent)
}

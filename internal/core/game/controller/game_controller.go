package controller

import (
	"github.com/LautaroBlasco23/impostor/internal/core/game/model"
	"github.com/LautaroBlasco23/impostor/internal/core/game/service"
	"github.com/gofiber/fiber/v2"
)

type GameController struct {
	service service.GameService
}

func NewGameController(service service.GameService) *GameController {
	return &GameController{service: service}
}

func (c *GameController) StartGame(ctx *fiber.Ctx) error {
	var req model.StartGameRequest
	if err := ctx.BodyParser(&req); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	game, err := c.service.StartGame(ctx.Context(), &req)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return ctx.Status(fiber.StatusCreated).JSON(game)
}

func (c *GameController) GetGame(ctx *fiber.Ctx) error {
	id := ctx.Params("id")

	game, err := c.service.GetGame(ctx.Context(), id)
	if err != nil {
		return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Game not found",
		})
	}

	return ctx.JSON(game)
}

func (c *GameController) GetGameByRoom(ctx *fiber.Ctx) error {
	roomID := ctx.Params("roomId")

	game, err := c.service.GetGameByRoom(ctx.Context(), roomID)
	if err != nil {
		return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "No active game in room",
		})
	}

	return ctx.JSON(game)
}

func (c *GameController) Vote(ctx *fiber.Ctx) error {
	var req model.VoteRequest
	if err := ctx.BodyParser(&req); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	result, err := c.service.Vote(ctx.Context(), &req)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return ctx.JSON(result)
}

func (c *GameController) EndGame(ctx *fiber.Ctx) error {
	id := ctx.Params("id")

	if err := c.service.EndGame(ctx.Context(), id); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return ctx.JSON(fiber.Map{
		"message": "Game ended",
	})
}

func (c *GameController) LeaveGame(ctx *fiber.Ctx) error {
	gameID := ctx.Params("id")

	var req model.LeaveGameRequest
	if err := ctx.BodyParser(&req); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if err := c.service.LeaveGame(ctx.Context(), gameID, &req); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return ctx.JSON(fiber.Map{
		"message": "Left game successfully",
	})
}

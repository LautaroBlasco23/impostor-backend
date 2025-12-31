package controller

import (
	"strconv"

	"github.com/LautaroBlasco23/impostor/internal/core/word/model"
	"github.com/LautaroBlasco23/impostor/internal/core/word/service"
	"github.com/gofiber/fiber/v2"
)

type WordController struct {
	service service.WordService
}

func NewWordController(service service.WordService) *WordController {
	return &WordController{service: service}
}

func (c *WordController) CreateWord(ctx *fiber.Ctx) error {
	var req model.CreateWordRequest
	if err := ctx.BodyParser(&req); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	word, err := c.service.CreateWord(ctx.Context(), &req)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return ctx.Status(fiber.StatusCreated).JSON(word)
}

func (c *WordController) GetWord(ctx *fiber.Ctx) error {
	id, err := strconv.Atoi(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid ID",
		})
	}

	word, err := c.service.GetWord(ctx.Context(), id)
	if err != nil {
		return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return ctx.JSON(word)
}

func (c *WordController) GetWordsByCategory(ctx *fiber.Ctx) error {
	category := ctx.Params("category")

	words, err := c.service.GetWordsByCategory(ctx.Context(), category)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return ctx.JSON(words)
}

func (c *WordController) GetRandomWords(ctx *fiber.Ctx) error {
	category := ctx.Params("category")
	limit := ctx.QueryInt("limit", 10)

	words, err := c.service.GetRandomWords(ctx.Context(), category, limit)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return ctx.JSON(words)
}

func (c *WordController) GetAllWords(ctx *fiber.Ctx) error {
	words, err := c.service.GetAllWords(ctx.Context())
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return ctx.JSON(words)
}

func (c *WordController) DeleteWord(ctx *fiber.Ctx) error {
	id, err := strconv.Atoi(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid ID",
		})
	}

	if err := c.service.DeleteWord(ctx.Context(), id); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return ctx.SendStatus(fiber.StatusNoContent)
}

package controller

import (
	"strconv"

	"github.com/LautaroBlasco23/impostor/internal/core/word/service"
	"github.com/gofiber/fiber/v2"
)

type WordController struct {
	service service.WordService
}

func NewWordController(service service.WordService) *WordController {
	return &WordController{service: service}
}

func (c *WordController) GetWord(ctx *fiber.Ctx) error {
	id, err := strconv.Atoi(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid ID",
		})
	}

	word, err := c.service.GetWord(ctx.UserContext(), id)
	if err != nil {
		return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
	return ctx.JSON(word)
}

func (c *WordController) GetWordsByCategory(ctx *fiber.Ctx) error {
	category := ctx.Params("category")

	words, err := c.service.GetWordsByCategory(ctx.UserContext(), category)
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

	words, err := c.service.GetRandomWords(ctx.UserContext(), category, limit)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
	return ctx.JSON(words)
}

func (c *WordController) GetAllWords(ctx *fiber.Ctx) error {
	words, err := c.service.GetAllWords(ctx.UserContext())
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
	return ctx.JSON(words)
}

func (c *WordController) GetCategories(ctx *fiber.Ctx) error {
	categories, err := c.service.GetCategories(ctx.UserContext())
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
	return ctx.JSON(categories)
}

func (c *WordController) DeleteWord(ctx *fiber.Ctx) error {
	id, err := strconv.Atoi(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid ID",
		})
	}

	if err := c.service.DeleteWord(ctx.UserContext(), id); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
	return ctx.SendStatus(fiber.StatusNoContent)
}

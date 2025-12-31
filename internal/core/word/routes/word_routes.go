package routes

import (
	"github.com/LautaroBlasco23/impostor/internal/core/word/controller"
	"github.com/gofiber/fiber/v2"
)

func RegisterRoutes(router fiber.Router, ctrl *controller.WordController) {
	router.Post("/", ctrl.CreateWord)
	router.Get("/", ctrl.GetAllWords)
	router.Get("/:id", ctrl.GetWord)
	router.Get("/category/:category", ctrl.GetWordsByCategory)
	router.Get("/category/:category/random", ctrl.GetRandomWords)
	router.Delete("/:id", ctrl.DeleteWord)
}

package routes

import (
	"github.com/LautaroBlasco23/impostor/internal/core/game/controller"
	"github.com/gofiber/fiber/v2"
)

func RegisterRoutes(router fiber.Router, ctrl *controller.GameController) {
	router.Post("/start", ctrl.StartGame)
	router.Get("/room/:roomId", ctrl.GetGameByRoom)
	router.Get("/:id", ctrl.GetGame)
	router.Post("/vote", ctrl.Vote)
	router.Post("/:id/end", ctrl.EndGame)
	router.Post("/:id/leave", ctrl.LeaveGame)
	router.Post("/:id/return", ctrl.ReturnToRoom)
}

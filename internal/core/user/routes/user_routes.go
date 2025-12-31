package routes

import (
	"github.com/LautaroBlasco23/impostor/internal/core/user/controller"
	"github.com/gofiber/fiber/v2"
)

func RegisterRoutes(router fiber.Router, ctrl *controller.UserController) {
	router.Post("/", ctrl.CreateUser)
	router.Get("/:id", ctrl.GetUser)
	router.Get("/room/:roomId", ctrl.GetUsersByRoom)
	router.Post("/:id/join", ctrl.JoinRoom)
	router.Post("/:id/ready", ctrl.ToggleReady)
	router.Delete("/:id", ctrl.DeleteUser)
}

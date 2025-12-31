package routes

import (
	"github.com/LautaroBlasco23/impostor/internal/core/room/controller"
	"github.com/gofiber/fiber/v2"
)

func RegisterRoutes(router fiber.Router, ctrl *controller.RoomController) {
	router.Post("/", ctrl.CreateRoom)
	router.Get("/", ctrl.GetAllRooms)
	router.Get("/:id", ctrl.GetRoom)
	router.Put("/:id/category", ctrl.SetCategory)
	router.Delete("/:id", ctrl.DeleteRoom)
}

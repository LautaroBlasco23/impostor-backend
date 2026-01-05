package main

import (
	"context"
	"errors"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/LautaroBlasco23/impostor/internal/config"
	gameController "github.com/LautaroBlasco23/impostor/internal/core/game/controller"
	gameRepo "github.com/LautaroBlasco23/impostor/internal/core/game/repository"
	gameRoutes "github.com/LautaroBlasco23/impostor/internal/core/game/routes"
	gameService "github.com/LautaroBlasco23/impostor/internal/core/game/service"
	roomController "github.com/LautaroBlasco23/impostor/internal/core/room/controller"
	roomRepo "github.com/LautaroBlasco23/impostor/internal/core/room/repository"
	roomRoutes "github.com/LautaroBlasco23/impostor/internal/core/room/routes"
	roomService "github.com/LautaroBlasco23/impostor/internal/core/room/service"
	userController "github.com/LautaroBlasco23/impostor/internal/core/user/controller"
	userRepo "github.com/LautaroBlasco23/impostor/internal/core/user/repository"
	userRoutes "github.com/LautaroBlasco23/impostor/internal/core/user/routes"
	userService "github.com/LautaroBlasco23/impostor/internal/core/user/service"
	wordController "github.com/LautaroBlasco23/impostor/internal/core/word/controller"
	wordRepo "github.com/LautaroBlasco23/impostor/internal/core/word/repository"
	wordRoutes "github.com/LautaroBlasco23/impostor/internal/core/word/routes"
	wordService "github.com/LautaroBlasco23/impostor/internal/core/word/service"
	"github.com/LautaroBlasco23/impostor/internal/database"
	"github.com/LautaroBlasco23/impostor/internal/middleware"
	ws "github.com/LautaroBlasco23/impostor/internal/websocket"
	wsController "github.com/LautaroBlasco23/impostor/internal/websocket/controller"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	cfg := config.Load()
	ctx := context.Background()

	redisClient, err := database.NewRedisClient(ctx, cfg.RedisURL)
	if err != nil {
		return err
	}
	defer func() {
		if err = redisClient.Close(); err != nil {
			log.Printf("redis close error: %v", err)
		}
	}()

	pgPool, err := database.NewPostgresPool(ctx, cfg.PostgresURL)
	if err != nil {
		return err
	}
	defer pgPool.Close()

	hub := ws.NewHub()
	go hub.Run()

	app := buildApp(cfg, pgPool, redisClient, hub)

	go func() {
		if err := app.Listen(":" + cfg.Port); err != nil {
			log.Printf("server error: %v", err)
		}
	}()

	return waitForShutdown(app)
}

func buildApp(
	cfg *config.Config,
	pgPool *pgxpool.Pool,
	redisClient *redis.Client,
	hub *ws.Hub,
) *fiber.App {
	wordRepository := wordRepo.NewWordRepository(pgPool)
	wordSvc := wordService.NewWordService(wordRepository)
	wordCtrl := wordController.NewWordController(wordSvc)

	roomRepository := roomRepo.NewRoomRepository(redisClient)
	roomSvc := roomService.NewRoomService(roomRepository, hub)
	roomCtrl := roomController.NewRoomController(roomSvc)

	userRepository := userRepo.NewUserRepository(redisClient)
	userSvc := userService.NewUserService(userRepository, hub)
	userCtrl := userController.NewUserController(userSvc)

	gameRepository := gameRepo.NewGameRepository(redisClient)
	gameSvc := gameService.NewGameService(
		gameRepository,
		roomRepository,
		userRepository,
		wordRepository,
		hub,
	)
	gameCtrl := gameController.NewGameController(gameSvc)

	wsCtrl := wsController.NewWebSocketController(hub)

	app := fiber.New(fiber.Config{
		AppName:      "Game Server",
		ErrorHandler: customErrorHandler,
	})

	app.Use(recover.New())
	app.Use(logger.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins: cfg.AllowedOrigins,
		AllowHeaders: "Origin, Content-Type, Accept, Authorization, Accept-Language",
		AllowMethods: "GET, POST, PUT, DELETE, OPTIONS",
	}))
	app.Use(middleware.Language())

	api := app.Group("/api/v1")
	wordRoutes.RegisterRoutes(api.Group("/words"), wordCtrl)
	roomRoutes.RegisterRoutes(api.Group("/rooms"), roomCtrl)
	userRoutes.RegisterRoutes(api.Group("/users"), userCtrl)
	gameRoutes.RegisterRoutes(api.Group("/games"), gameCtrl)

	app.Use("/ws", wsCtrl.UpgradeMiddleware)
	app.Get("/ws/:userId", websocket.New(wsCtrl.HandleConnection))

	return app
}

func waitForShutdown(app *fiber.App) error {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	return app.ShutdownWithContext(ctx)
}

func customErrorHandler(c *fiber.Ctx, err error) error {
	code := fiber.StatusInternalServerError

	var e *fiber.Error
	if errors.As(err, &e) {
		code = e.Code
	}

	return c.Status(code).JSON(fiber.Map{
		"error": err.Error(),
	})
}

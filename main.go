package main

import (
	"fmt"
	"time"

	"github.com/DraftTin/URLShortener-go-fiber-redis/api/routes"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"

	// "github.com/joho/godotenv"
	"log"
	// "os"
)

func setupRoutes(app *fiber.App) {
	// app.Get("/:url", routes.ResolveURL)
	app.Post("/api/v1", routes.ShortenURL)
}

func main() {
	fmt.Println(time.Now())
	// err := godotenv.Load()
	// if err != nil {
	// 	panic(err)
	// }
	app := fiber.New()

	app.Use(logger.New())

	setupRoutes(app)
	// log.Fatal(app.Listen(os.Getenv("APP_PORT")))
	log.Fatal(app.Listen(":3000"))
}

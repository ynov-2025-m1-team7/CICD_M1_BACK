package main

import (
	"github.com/gofiber/fiber/v2"
)

func main() {
	app := fiber.New()

	// Define a basic route
	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("Welcome to the Fiber RESTful API!")
	})

	// Start the server
	app.Listen(":8080")
}

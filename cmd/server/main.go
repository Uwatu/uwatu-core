package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/gofiber/fiber/v2"
	"github.com/uwatu/uwatu-core/internal/ingestion" // <-- Check go.mod for your true module path
	"github.com/uwatu/uwatu-core/internal/nokia"
)

func main() {
	log.Println("========================================")
	log.Println("  Uwatu Core API Gateway")
	log.Println("========================================")

	// 1. Build Nokia Client (Use fake credentials until Nokia gives you real ones)
	nokiaClient := nokia.NewClient(
		"fake_id",
		"fake_secret",
		"https://sandbox.networkascode.nokia.io",
	)

	// 2. Build the Enricher
	enricher := ingestion.NewEnricher(nokiaClient)

	// 3. Build and Start the MQTT Listener in the background
	mqttHandler := ingestion.NewHandler(enricher)
	go mqttHandler.StartMQTT("tcp://broker.hivemq.com:1883", "uwatu_core_elvis")

	// 4. Start Fiber Web Server
	app := fiber.New(fiber.Config{DisableStartupMessage: true})

	app.Get("/health", func(c *fiber.Ctx) error {
		return c.SendString("Uwatu Core OK!")
	})

	go func() {
		log.Println("[SERVER] Fiber listening on port 8080")
		if err := app.Listen(":8080"); err != nil {
			log.Panic(err)
		}
	}()

	// 5. Keep running until Ctrl+C is pressed
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down Uwatu Core...")
}

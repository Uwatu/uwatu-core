package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/gofiber/fiber/v2"
	"github.com/uwatu/uwatu-core/internal/config"
	"github.com/uwatu/uwatu-core/internal/ingestion"
	"github.com/uwatu/uwatu-core/internal/nokia"
)

func main() {
	config.LogInfo("SYSTEM", "Starting Uwatu Core API Gateway...")

	nokiaClient := nokia.NewClient("fake_id", "fake_secret", "https://sandbox.networkascode.nokia.io")
	enricher := ingestion.NewEnricher(nokiaClient)
	mqttHandler := ingestion.NewHandler(enricher)

	go mqttHandler.StartMQTT("tcp://broker.hivemq.com:1883", "uwatu_core_elvis")

	app := fiber.New(fiber.Config{DisableStartupMessage: true})

	go func() {
		config.LogInfo("SERVER", "Fiber listening on port 8080")
		if err := app.Listen(":8080"); err != nil {
			config.LogError("SERVER", err.Error())
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	config.LogInfo("SYSTEM", "Shutting down...")
}

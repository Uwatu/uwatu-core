package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"firebase.google.com/go/v4/messaging"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
	"github.com/uwatu/uwatu-core/internal/alerts"
	"github.com/uwatu/uwatu-core/internal/config"
	"github.com/uwatu/uwatu-core/internal/db"
	"github.com/uwatu/uwatu-core/internal/farm"
	"github.com/uwatu/uwatu-core/internal/geofence"
	"github.com/uwatu/uwatu-core/internal/ingestion"
	"github.com/uwatu/uwatu-core/internal/nokia"
	"github.com/uwatu/uwatu-core/internal/ws"
)

func main() {
	config.LogInfo("SYSTEM", "Starting Uwatu Core API Gateway...")

	// Optional: load structured config (non-fatal)
	cfg, err := config.LoadConfig(".")
	if err != nil {
		config.LogError("CONFIG", "Failed to load config: "+err.Error())
	}

	// Database
	if err := db.Connect(); err != nil {
		config.LogError("DB", err.Error())
	}
	db.RunMigrations()
	defer db.Close()

	// Farm and animal registries
	farmReg := farm.NewRegistry(db.Pool)
	animalReg := farm.NewAnimalRegistry(db.Pool)

	// Geofence manager (loads boundaries from DB)
	geofenceMgr := geofence.NewManager(db.Pool)
	if err := geofenceMgr.LoadAll(context.Background()); err != nil {
		config.LogError("GEOFENCE", "Failed to load geofences: "+err.Error())
	}

	// Nokia client
	nokiaClient := nokia.NewClient(
		os.Getenv("NOKIA_RAPIDAPI_KEY"),
		"network-as-code.nokia.rapidapi.com",
		"https://network-as-code.p-eu.rapidapi.com",
	)

	// Firebase (optional)
	var fcmClient *messaging.Client
	if cfg.FirebaseProjectID != "" {
		fcmClient, _ = alerts.InitializeFCM(cfg.FirebaseProjectID)
	}

	// Alert router
	router := alerts.NewAlertRouter(
		db.Pool,
		fcmClient,
		os.Getenv("AT_API_KEY"),
		os.Getenv("AT_SANDBOX_USERNAME"),
		"UWATU",
	)

	// WebSocket hub
	hub := ws.NewHub()

	// Enricher with full dependencies
	enricher := ingestion.NewEnricher(nokiaClient, animalReg, farmReg, geofenceMgr, router, hub)

	// MQTT handler
	mqttHandler := ingestion.NewHandler(enricher)
	go mqttHandler.StartMQTT("tcp://broker.hivemq.com:1883", fmt.Sprintf("uwatu_core_%d", time.Now().Unix()))

	// Fiber app
	app := fiber.New(fiber.Config{DisableStartupMessage: true})

	// ---- Geofence onboarding endpoint ----
	app.Post("/api/farms", func(c *fiber.Ctx) error {
		var body struct {
			FarmName string        `json:"farmName"`
			Boundary [][][]float64 `json:"boundary"`
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": err.Error()})
		}
		if len(body.Boundary) == 0 || len(body.Boundary[0]) < 3 {
			return c.Status(400).JSON(fiber.Map{"error": "boundary must be a polygon with at least 3 points"})
		}

		wkt := "POLYGON(("
		for i, coord := range body.Boundary[0] {
			if i > 0 {
				wkt += ","
			}
			wkt += fmt.Sprintf("%f %f", coord[0], coord[1])
		}
		wkt += "))"

		_, err := db.Pool.Exec(context.Background(),
			"INSERT INTO farms (id, farmer_id, name, boundary) VALUES (gen_random_uuid(), 'demo-farmer', $1, ST_GeomFromText($2, 4326))",
			body.FarmName, wkt)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}

		if err := geofenceMgr.LoadAll(context.Background()); err != nil {
			config.LogError("GEOFENCE", "Failed to reload geofences: "+err.Error())
		}
		config.LogInfo("GEOFENCE", fmt.Sprintf("Boundary stored for farm: %s", body.FarmName))
		return c.JSON(fiber.Map{"status": "ok", "farmName": body.FarmName})
	})

	// ---- WebSocket ----
	app.Use("/ws", func(c *fiber.Ctx) error {
		if websocket.IsWebSocketUpgrade(c) {
			return c.Next()
		}
		return fiber.ErrUpgradeRequired
	})
	app.Get("/ws/farm/:farm_id", websocket.New(func(c *websocket.Conn) {
		hub.Upgrade(c)
	}))

	// Start server
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

package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"
	//"github.com/gorilla/websocket"
	"firebase.google.com/go/v4/messaging"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
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
	app := fiber.New()

	// ---- CORS ----
	app.Use(cors.New(cors.Config{
		AllowOrigins:     "http://localhost:3000", // frontend origin
		AllowMethods:     "GET,POST,PUT,DELETE,OPTIONS",
		AllowHeaders:     "Content-Type,Authorization",
		AllowCredentials: true,
	}))

	// ---- Registration Endpoint ----
	app.Post("/api/register", func(c *fiber.Ctx) error {
		var body struct {
			Phone      string `json:"phone"`
			Name       string `json:"name"`
			Locale     string `json:"locale"`      // optional, default "en"
			DeviceTier int    `json:"device_tier"` // optional, default 1
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
		}
		if body.Phone == "" || body.Name == "" {
			return c.Status(400).JSON(fiber.Map{"error": "phone and name are required"})
		}
		if body.Locale == "" {
			body.Locale = "en"
		}
		if body.DeviceTier == 0 {
			body.DeviceTier = 1
		}

		// Use demo-farmer ID (hackathon shortcut)
		farmerID := "demo-farmer"

		// Upsert farmer
		_, err := db.Pool.Exec(context.Background(),
			`INSERT INTO farmers (id, name, phone, device_tier, locale)
             VALUES ($1, $2, $3, $4, $5)
             ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name, phone = EXCLUDED.phone`,
			farmerID, body.Name, body.Phone, body.DeviceTier, body.Locale)
		if err != nil {
			config.LogError("REGISTER", "failed to upsert farmer: "+err.Error())
			return c.Status(500).JSON(fiber.Map{"error": "registration failed"})
		}

		config.LogInfo("REGISTER", fmt.Sprintf("Farmer registered: id=%s phone=%s name=%s", farmerID, body.Phone, body.Name))
		return c.JSON(fiber.Map{
			"farmer_id": farmerID,
			"status":    "ok",
		})
	})

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
			wkt += fmt.Sprintf("%f %f", coord[0], coord[1]) // lon lat
		}
		wkt += "))"

		farmID := "demo-farm" // fixed ID for hackathon

		_, err := db.Pool.Exec(context.Background(),
			`INSERT INTO farms (id, farmer_id, name, boundary)
         VALUES ($1, 'demo-farmer', $2, ST_GeomFromText($3, 4326))
         ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name, boundary = EXCLUDED.boundary`,
			farmID, body.FarmName, wkt)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}

		// Reload geofence cache
		if err := geofenceMgr.LoadAll(context.Background()); err != nil {
			config.LogError("GEOFENCE", "failed to reload: "+err.Error())
		}
		return c.JSON(fiber.Map{"status": "ok", "farmId": farmID})
	})

	// ---- Get farm boundary (by farm ID) ----
	app.Get("/api/farms/:farmID", func(c *fiber.Ctx) error {
		farmID := c.Params("farmID")
		var name string
		var wkt string
		err := db.Pool.QueryRow(context.Background(),
			"SELECT name, ST_AsText(boundary) FROM farms WHERE id = $1", farmID,
		).Scan(&name, &wkt)
		if err != nil {
			return c.Status(404).JSON(fiber.Map{"error": "farm not found"})
		}

		// Parse WKT polygon into coordinates
		var coords [][][]float64
		wkt = strings.TrimPrefix(wkt, "POLYGON((")
		wkt = strings.TrimSuffix(wkt, "))")
		rings := strings.Split(wkt, "),(")
		for _, ring := range rings {
			points := strings.Split(ring, ",")
			var ringCoords [][]float64
			for _, p := range points {
				parts := strings.Split(strings.TrimSpace(p), " ")
				if len(parts) == 2 {
					lon, _ := strconv.ParseFloat(parts[0], 64)
					lat, _ := strconv.ParseFloat(parts[1], 64)
					ringCoords = append(ringCoords, []float64{lon, lat})
				}
			}
			coords = append(coords, ringCoords)
		}

		return c.JSON(fiber.Map{
			"farmId":   farmID,
			"farmName": name,
			"boundary": coords,
		})
	})

	// ---- OTP Verification Stub (hackathon) ----
	app.Post("/api/verify-otp", func(c *fiber.Ctx) error {
		// Always succeed – real verification would check code via SMS gateway
		return c.JSON(fiber.Map{"verified": true})
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

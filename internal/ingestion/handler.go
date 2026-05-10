package ingestion

import (
	"encoding/json"
	"fmt"
	"log"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/uwatu/uwatu-core/internal/config"
	"github.com/uwatu/uwatu-core/internal/models"
)

type Handler struct {
	enricher *Enricher
}

func NewHandler(e *Enricher) *Handler {
	return &Handler{enricher: e}
}

func (h *Handler) StartMQTT(brokerURL string, clientID string) {
	opts := mqtt.NewClientOptions()
	opts.AddBroker(brokerURL)
	opts.SetClientID(clientID)
	opts.SetDefaultPublishHandler(h.handleMessage)

	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		log.Fatalf("[FATAL] MQTT Connect failed: %v", token.Error())
	}

	client.Subscribe("uwatu/farm/+/tag/+", 1, nil)
	log.Printf("[INGESTION] Listening on HiveMQ for cow data...")
}

func (h *Handler) handleMessage(client mqtt.Client, msg mqtt.Message) {
	var payload map[string]interface{}
	if err := json.Unmarshal(msg.Payload(), &payload); err != nil {
		config.LogError("MQTT", "Failed to decode JSON payload")
		return
	}

	deviceID, _ := payload["device_id"].(string)
	msisdn, _ := payload["msisdn"].(string)

	telemetry := models.TagTelemetry{}
	if firmware, ok := payload["firmware_payload"].(map[string]interface{}); ok {
		if v, ok := firmware["body_temp_c"].(float64); ok {
			telemetry.BodyTempC = v
		}
		if v, ok := firmware["accel_magnitude"].(float64); ok {
			telemetry.AccelMagnitude = int(v)
		}
		if v, ok := firmware["battery_pct"].(float64); ok {
			telemetry.BatteryPct = int(v)
		}
	} else {
		config.LogError("INGEST", "Message arrived but firmware_payload was missing!")
	}

	// ── Simulator overrides ──────────────────────────────────────
	// SIM swap: default false
	simSwapDefault := false
	simSwapOverride := &simSwapDefault
	if v, ok := payload["sim_swap"].(bool); ok {
		simSwapOverride = &v
	}

	// Location: only when simulator gives coordinates
	var demoLat, demoLon *float64
	if v, ok := payload["demo_lat"].(float64); ok {
		demoLat = &v
	}
	if v, ok := payload["demo_lon"].(float64); ok {
		demoLon = &v
	}

	// Roaming: default false
	roamingDefault := false
	roamingOverride := &roamingDefault
	if v, ok := payload["roaming"].(bool); ok {
		roamingOverride = &v
	}

	// Device swap: default false
	deviceSwapDefault := false
	deviceSwapOverride := &deviceSwapDefault

	// Connectivity: default false (meaning reachable)
	connectivityDefault := false
	connectivityOverride := &connectivityDefault
	if v, ok := payload["connectivity"].(bool); ok {
		connectivityOverride = &v
	}

	config.LogInfo("INGEST", fmt.Sprintf("Tag: %s | Signal: Recv", deviceID))

	h.enricher.Process(deviceID, msisdn, telemetry,
		simSwapOverride, demoLat, demoLon,
		roamingOverride, deviceSwapOverride, connectivityOverride)
}

package ingestion

import (
	"encoding/json"
	"fmt"
	"log"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/uwatu/uwatu-core/internal/config"
	"github.com/uwatu/uwatu-core/internal/models"
)

// Handler receives MQTT messages and forwards parsed telemetry to the Enricher.
type Handler struct {
	enricher *Enricher
}

// NewHandler creates a new Handler with a reference to the Enricher.
func NewHandler(e *Enricher) *Handler {
	return &Handler{enricher: e}
}

// StartMQTT connects to the broker and subscribes to the telemetry topic.
func (h *Handler) StartMQTT(brokerURL string, clientID string) {
	opts := mqtt.NewClientOptions()
	opts.AddBroker(brokerURL)
	opts.SetClientID(clientID)

	// When a message arrives, run handleMessage
	opts.SetDefaultPublishHandler(h.handleMessage)

	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		log.Fatalf("[FATAL] MQTT Connect failed: %v", token.Error())
	}

	// Listen to all tags on all farms
	client.Subscribe("uwatu/farm/+/tag/+", 1, nil)
	log.Printf("[INGESTION] Listening on HiveMQ for cow data...")
}

// handleMessage unpacks the simulator JSON payload, builds a TagTelemetry,
// and passes it to the Enricher.
func (h *Handler) handleMessage(client mqtt.Client, msg mqtt.Message) {
	var payload map[string]interface{}
	if err := json.Unmarshal(msg.Payload(), &payload); err != nil {
		config.LogError("MQTT", "Failed to decode JSON payload")
		return
	}

	// 1. Extract top-level identifiers
	deviceID, _ := payload["device_id"].(string)
	msisdn, _ := payload["msisdn"].(string)

	// 2. Build TagTelemetry from the nested firmware_payload
	telemetry := models.TagTelemetry{}
	if firmware, ok := payload["firmware_payload"].(map[string]interface{}); ok {
		// JSON numbers are float64 in Go – convert to the correct type
		if v, ok := firmware["body_temp_c"].(float64); ok {
			telemetry.BodyTempC = v
		}
		if v, ok := firmware["accel_magnitude"].(float64); ok {
			telemetry.AccelMagnitude = int(v)
		}
		if v, ok := firmware["battery_pct"].(float64); ok {
			telemetry.BatteryPct = int(v)
		}
		// Additional fields (battery_mv, rssi_dbm, cell_id, etc.) can be
		// mapped as the simulator begins to populate them.
	} else {
		config.LogError("INGEST", "Message arrived but firmware_payload was missing!")
	}

	config.LogInfo("INGEST", fmt.Sprintf("Tag: %s | Signal: Recv", deviceID))

	// 3. Hand over to the enricher
	h.enricher.Process(deviceID, msisdn, telemetry)
}

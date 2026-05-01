package ingestion

import (
	"encoding/json"
	"fmt"
	"log"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/uwatu/uwatu-core/internal/config"
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

	// When a message arrives, run handleMessage
	opts.SetDefaultPublishHandler(h.handleMessage)

	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		log.Fatalf("[FATAL] MQTT Connect failed: %v", token.Error())
	}

	// Listen to all tags on all farms (+ is a wildcard)
	client.Subscribe("uwatu/farm/+/tag/+", 1, nil)
	log.Printf("[INGESTION] Listening on HiveMQ for cow data...")
}

func (h *Handler) handleMessage(client mqtt.Client, msg mqtt.Message) {
	var payload map[string]interface{}
	if err := json.Unmarshal(msg.Payload(), &payload); err != nil {
		config.LogError("MQTT", "Failed to decode JSON payload")
		return
	}

	// 1. Get the Top-Level fields
	deviceID, _ := payload["device_id"].(string)
	msisdn, _ := payload["msisdn"].(string)

	// 2. Open the "firmware_payload" sub-folder
	// In Go, we have to "type assert" it to another map
	firmware, ok := payload["firmware_payload"].(map[string]interface{})

	// Create variables to hold our data (default to 0)
	var temp float64
	var accel int
	var battery int

	if ok {
		// 3. Extract the data from INSIDE the firmware_payload
		// Note: JSON numbers are ALWAYS float64 in Go, so we convert them
		temp, _ = firmware["body_temp_c"].(float64)

		accelVal, _ := firmware["accel_magnitude"].(float64)
		accel = int(accelVal)

		battVal, _ := firmware["battery_pct"].(float64)
		battery = int(battVal)
	} else {
		config.LogError("INGEST", "Message arrived but firmware_payload was missing!")
	}

	config.LogInfo("INGEST", fmt.Sprintf("Tag: %s | Signal: Recv", deviceID))

	// 4. Pass the data to the enricher
	h.enricher.Process(deviceID, msisdn, battery, temp, accel)
}

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
	// Unpack the cow's JSON message
	var payload map[string]interface{}
	if err := json.Unmarshal(msg.Payload(), &payload); err != nil {
		return
	}

	// Extract the important bits
	deviceID, _ := payload["device_id"].(string)
	msisdn, _ := payload["msisdn"].(string)

	// Battery is tricky in JSON, it comes through as a float64
	batteryFloat, _ := payload["battery_pct"].(float64)
	battery := int(batteryFloat)

	temp, _ := payload["body_temp_c"].(float64)
	accel, _ := payload["accel_magnitude"].(float64)

	// Log the receipt with a clean string
	config.LogInfo("INGEST", fmt.Sprintf("Tag: %s | Signal: Recv", deviceID))

	h.enricher.Process(deviceID, msisdn, int(battery), temp, int(accel))

}

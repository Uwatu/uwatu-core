package alerts

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// WhatsAppPayload represents the JSON body required by AT's WhatsApp API.
type WhatsAppPayload struct {
	Username string `json:"username"`
	To       string `json:"to"`
	From     string `json:"from"`
	Message  string `json:"message"`
}

// SendWhatsApp dispatches a WhatsApp message via the Africa's Talking API.
func SendWhatsApp(apiKey string, username string, from string, to string, message string) error {
	// The specific endpoint for Africa's Talking Sandbox WhatsApp
	endpoint := "https://api.sandbox.africastalking.com/whatsapp/message/send"

	payload := WhatsAppPayload{
		Username: username,
		From:     from,
		To:       to,
		Message:  message,
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("error while marshalling whatsapp payload: %w", err)
	}

	whatsAppRequest, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return fmt.Errorf("error while creating whatsapp request: %w", err)
	}

	whatsAppRequest.Header.Set("Content-Type", "application/json")
	whatsAppRequest.Header.Set("apiKey", apiKey)

	client := &http.Client{}
	resp, err := client.Do(whatsAppRequest)
	if err != nil {
		return fmt.Errorf("failed to execute whatsapp request: %w", err)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {

		}
	}(resp.Body)

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("whatsapp api rejected request, status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	return nil
}

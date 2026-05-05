package alerts

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type WhatsAppBody struct {
	Message string `json:"message"`
}

type WhatsAppPayload struct {
	Username    string       `json:"username"`
	WaNumber    string       `json:"waNumber"`    // Changed to match documentation
	PhoneNumber string       `json:"phoneNumber"` // Changed to match documentation
	Body        WhatsAppBody `json:"body"`        // Documentation requires this nested object
}

func SendWhatsApp(apiKey string, username string, from string, to string, message string) error {
	// Sandbox is "Coming Soon" per AT docs — this hits live
	endpoint := "https://chat.africastalking.com/whatsapp/message/send"

	payload := WhatsAppPayload{
		Username:    username,
		WaNumber:    from,
		PhoneNumber: to,
		Body: WhatsAppBody{
			Message: message,
		},
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("error while marshalling whatsapp payload: %w", err)
	}

	req, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return fmt.Errorf("error while creating whatsapp request: %w", err)
	}

	// Use exact header casing from documentation
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("apiKey", apiKey)

	resp, err := AlertHTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute whatsapp request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("whatsapp api rejected request, status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	return nil
}

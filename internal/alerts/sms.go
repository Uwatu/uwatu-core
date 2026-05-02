package alerts

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// SendSMS dispatches a text message via the Africa's Talking API.
func SendSMS(apiKey string, username string, to string, message string) error {
	// The specific endpoint for Africa's Talking Sandbox SMS
	endpoint := "https://api.sandbox.africastalking.com/version1/messaging"

	data := url.Values{}
	data.Set("username", username)

	data.Set("to", to)
	data.Set("message", message)

	// Create the HTTP request
	req, err := http.NewRequest("POST", endpoint, strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create sms request: %w", err)
	}

	req.Header.Set("apiKey", apiKey)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := AlertHTTPClient.Do(req)

	if err != nil {
		return fmt.Errorf("failed to send sms request: %w", err)
	}

	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("failed to send sms request, got status code %d: %s", resp.StatusCode, string(bodyBytes))
	}

	return nil
}

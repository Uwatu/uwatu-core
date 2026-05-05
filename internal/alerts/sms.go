package alerts

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

func SendSMS(apiKey string, username string, to string, message string) error {
	endpoint := "https://api.sandbox.africastalking.com/version1/messaging"

	formData := url.Values{}
	formData.Set("username", username)
	formData.Set("message", message)
	formData.Set("to", to)

	req, err := http.NewRequest("POST", endpoint, strings.NewReader(formData.Encode()))
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

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("sms api error, status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	return nil
}

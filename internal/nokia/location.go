package nokia

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// LocationResult is how Nokia formats their answer
type LocationResult struct {
	Lat float64 `json:"latitude"`
	Lon float64 `json:"longitude"`
}

func (c *Client) GetDeviceLocation(ctx context.Context, msisdn string) (*LocationResult, error) {
	// Rule: Never wait more than 3 seconds for Nokia
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	// Wait for the green light from our 100 req/min rate limiter
	c.rateLimiter.Wait(ctx)

	// Build the JSON message Nokia expects
	body := []byte(fmt.Sprintf(`{"device":{"phoneNumber":"%s"},"maxAge":60}`, msisdn))

	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/location-retrieval/v0/retrieve", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	// Send the message
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Unpack Nokia's answer
	var result LocationResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result, nil
}

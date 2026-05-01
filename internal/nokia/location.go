package nokia

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/uwatu/uwatu-core/internal/config"
)

// LocationResult maps the expected JSON response structure from the
// Nokia Network as Code Location Retrieval API.
type LocationResult struct {
	LastLocation string `json:"lastLocationTime"`
	Area         struct {
		AreaType string `json:"areaType"`
		Center   struct {
			Lat float64 `json:"latitude"`
			Lon float64 `json:"longitude"`
		} `json:"center"`
		Radius int `json:"radius"`
	} `json:"area"`
}

// GetDeviceLocation retrieves the network-topology-derived coordinates for a SIM.
// It enforces a 3-second context timeout and utilizes the shared RapidAPI client.
// Verification logs are emitted to the terminal to trace active network requests.
func (c *Client) GetDeviceLocation(ctx context.Context, msisdn string) (*LocationResult, error) {
	// Enforce architectural deadline for external network calls
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	// Wait for rate limiter to ensure compliance with 100 req/min limit
	if err := c.rateLimiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limit exceeded: %w", err)
	}

	// Construct CAMARA-compliant request payload
	requestBody, err := json.Marshal(map[string]interface{}{
		"device": map[string]string{
			"phoneNumber": msisdn,
		},
		"maxAge": 60,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Initialize POST request to the location retrieval endpoint
	endpoint := c.baseURL + "/location-retrieval/v0/retrieve"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(requestBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Log outbound request for verification of live network activity
	config.LogInfo("HTTP", fmt.Sprintf("--> POST /location-retrieval/v0/retrieve [MSISDN: %s]", msisdn))

	// Execute request using the RapidAPI helper to inject authentication headers
	resp, err := c.DoRequest(req)
	if err != nil {
		config.LogError("HTTP", fmt.Sprintf("--> request failed: %v", err))
		return nil, fmt.Errorf("http request failed: %w", err)
	}

	// Capture tracing headers from Nokia/RapidAPI for audit trails
	requestID := resp.Header.Get("X-Request-Id")

	// Handle non-200 status codes from the API gateway
	if resp.StatusCode != http.StatusOK {
		config.LogError("HTTP", fmt.Sprintf("<-- %d Error [RequestID: %s]", resp.StatusCode, requestID))
		return nil, fmt.Errorf("nokia api error: status %d", resp.StatusCode)
	}

	// Log inbound success to confirm receipt of external data
	config.LogInfo("HTTP", fmt.Sprintf("<-- 200 OK [RequestID: %s]", requestID))

	// Decode the JSON response into the result struct
	var result LocationResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	return &result, nil
}

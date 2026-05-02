package nokia

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/uwatu/uwatu-core/internal/config"
)

// DeviceStatusResult maps the response from the Device Reachability Status Retrieve v1 API.
type DeviceStatusResult struct {
	Reachable    bool     `json:"reachable"`
	Connectivity []string `json:"connectivity"` // e.g. ["SMS"] or ["DATA"]
	LastStatusAt string   `json:"lastStatusTime"`
}

// GetDeviceStatus queries the reachability status of a device.
// Returns the raw result including whether the device is reachable and how (SMS, DATA).
func (c *Client) GetDeviceStatus(ctx context.Context, msisdn string) (*DeviceStatusResult, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	if err := c.rateLimiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limit exceeded: %w", err)
	}

	// Build request body
	body := map[string]interface{}{
		"device": map[string]string{
			"phoneNumber": msisdn,
		},
	}
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	endpoint := c.baseURL + "/device-status/device-reachability-status/v1/retrieve"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	config.LogInfo("HTTP", fmt.Sprintf("--> POST /device-status/device-reachability-status/v1/retrieve [MSISDN: %s]", msisdn))

	resp, err := c.DoRequest(req)
	if err != nil {
		config.LogError("HTTP", fmt.Sprintf("--> request failed: %v", err))
		return nil, fmt.Errorf("status http request: %w", err)
	}
	defer resp.Body.Close()

	requestID := resp.Header.Get("X-Request-Id")
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		config.LogError("HTTP", fmt.Sprintf("<-- %d Error [RequestID: %s] | Body: %s", resp.StatusCode, requestID, string(bodyBytes)))
		return nil, fmt.Errorf("device status api error: status %d", resp.StatusCode)
	}

	config.LogInfo("HTTP", fmt.Sprintf("<-- 200 OK [RequestID: %s]", requestID))

	var result DeviceStatusResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode device status: %w", err)
	}
	return &result, nil
}

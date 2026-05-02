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

// DeviceSwapResult holds the response from the Device Swap check.
type DeviceSwapResult struct {
	Swapped bool `json:"swapped"`
}

// CheckDeviceSwap tells whether the device's IMEI has changed within the give maxAge (hours).
func (c *Client) CheckDeviceSwap(ctx context.Context, msisdn string, maxAge int) (*DeviceSwapResult, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	if err := c.rateLimiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limit exceeded: %w", err)
	}

	reqBody, err := json.Marshal(map[string]interface{}{
		"phoneNumber": msisdn,
		"maxAge":      maxAge,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	endpoint := c.baseURL + "/passthrough/camara/v1/device-swap/device-swap/v1/check"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	config.LogInfo("HTTP", fmt.Sprintf("--> POST /passthrough/camara/v1/device-swap/device-swap/v1/check [MSISDN: %s]", msisdn))

	resp, err := c.DoRequest(req)
	if err != nil {
		config.LogError("HTTP", fmt.Sprintf("--> request failed: %v", err))
		return nil, fmt.Errorf("device swap request: %w", err)
	}
	defer resp.Body.Close()

	requestID := resp.Header.Get("X-Request-Id")
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		config.LogError("HTTP", fmt.Sprintf("<-- %d Error [RequestID: %s] | Body: %s", resp.StatusCode, requestID, string(bodyBytes)))
		return nil, fmt.Errorf("device swap api error: status %d", resp.StatusCode)
	}

	config.LogInfo("HTTP", fmt.Sprintf("<-- 200 OK [RequestID: %s]", requestID))

	var result DeviceSwapResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode device swap response: %w", err)
	}
	return &result, nil
}

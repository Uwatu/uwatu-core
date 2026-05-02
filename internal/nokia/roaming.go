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

// RoamingResult holds the response from the Device Roaming Status API.
type RoamingResult struct {
	Roaming     bool `json:"roaming"`
	CountryCode int  `json:"countryCode"` // numeric country code, e.g. 159 for ??
}

// GetRoamingStatus checks whether the device is currently roaming and in which country.
func (c *Client) GetRoamingStatus(ctx context.Context, msisdn string) (*RoamingResult, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	if err := c.rateLimiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limit exceeded: %w", err)
	}

	body := map[string]interface{}{
		"device": map[string]string{
			"phoneNumber": msisdn,
		},
	}
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	endpoint := c.baseURL + "/device-status/device-roaming-status/v1/retrieve"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	config.LogInfo("HTTP", fmt.Sprintf("--> POST /device-status/device-roaming-status/v1/retrieve [MSISDN: %s]", msisdn))

	resp, err := c.DoRequest(req)
	if err != nil {
		config.LogError("HTTP", fmt.Sprintf("--> request failed: %v", err))
		return nil, fmt.Errorf("roaming http request: %w", err)
	}
	defer resp.Body.Close()

	requestID := resp.Header.Get("X-Request-Id")
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		config.LogError("HTTP", fmt.Sprintf("<-- %d Error [RequestID: %s] | Body: %s", resp.StatusCode, requestID, string(bodyBytes)))
		return nil, fmt.Errorf("roaming api error: status %d", resp.StatusCode)
	}

	config.LogInfo("HTTP", fmt.Sprintf("<-- 200 OK [RequestID: %s]", requestID))

	var result RoamingResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode roaming response: %w", err)
	}
	return &result, nil
}

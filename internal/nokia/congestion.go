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

// CongestionResult maps the first element of the congestion fetch response.
type CongestionResult struct {
	CongestionLevel   string `json:"congestionLevel"` // "Low", "Medium", "High"
	ConfidenceLevel   int    `json:"confidenceLevel"`
	TimeIntervalStart string `json:"timeIntervalStart"`
	TimeIntervalStop  string `json:"timeIntervalStop"`
}

// GetCongestion fetches the current congestion level for the cell serving the device.
func (c *Client) GetCongestion(ctx context.Context, msisdn string) (*CongestionResult, error) {
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

	endpoint := c.baseURL + "/congestion-insights/v0/fetch"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	config.LogInfo("HTTP", fmt.Sprintf("--> POST /congestion-insights/v0/fetch [MSISDN: %s]", msisdn))

	resp, err := c.DoRequest(req)
	if err != nil {
		config.LogError("HTTP", fmt.Sprintf("--> request failed: %v", err))
		return nil, fmt.Errorf("congestion http request: %w", err)
	}
	defer resp.Body.Close()

	requestID := resp.Header.Get("X-Request-Id")
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		config.LogError("HTTP", fmt.Sprintf("<-- %d Error [RequestID: %s] | Body: %s", resp.StatusCode, requestID, string(bodyBytes)))
		return nil, fmt.Errorf("congestion api error: status %d", resp.StatusCode)
	}

	config.LogInfo("HTTP", fmt.Sprintf("<-- 200 OK [RequestID: %s]", requestID))

	// The response is an array – unmarshal as slice and take the first element.
	var raw []CongestionResult
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("failed to decode congestion response: %w", err)
	}
	if len(raw) == 0 {
		return &CongestionResult{}, nil // no data, return zero value
	}
	return &raw[0], nil
}

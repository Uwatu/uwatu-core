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

// SIMSwapResult maps the Nokia SIM Swap check response.
type SIMSwapResult struct {
	Swapped bool `json:"swapped"`
}

// CheckSIMSwap queries whether a SIM swap has occurred within the given maxAge (hours).
func (c *Client) CheckSIMSwap(ctx context.Context, msisdn string) (*SIMSwapResult, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	if err := c.rateLimiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limit exceeded: %w", err)
	}

	requestBody, err := json.Marshal(map[string]interface{}{
		"phoneNumber": msisdn,
		"maxAge":      240,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Actual passthrough endpoint from the Nokia Marketplace
	endpoint := c.baseURL + "/passthrough/camara/v1/sim-swap/sim-swap/v0/check"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(requestBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	config.LogInfo("HTTP", fmt.Sprintf("--> POST /passthrough/camara/v1/sim-swap/sim-swap/v0/check [MSISDN: %s]", msisdn))

	resp, err := c.DoRequest(req)
	if err != nil {
		config.LogError("HTTP", fmt.Sprintf("--> request failed: %v", err))
		return nil, fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()

	requestID := resp.Header.Get("X-Request-Id")
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		config.LogError("HTTP", fmt.Sprintf("<-- %d Error [RequestID: %s] | Body: %s", resp.StatusCode, requestID, string(bodyBytes)))
		return nil, fmt.Errorf("nokia sim-swap error: %d", resp.StatusCode)
	}

	config.LogInfo("HTTP", fmt.Sprintf("<-- 200 OK [RequestID: %s]", requestID))

	var result SIMSwapResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode sim-swap response: %w", err)
	}
	return &result, nil
}

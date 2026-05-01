package nokia

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type SIMSwapResult struct {
	Swapped bool `json:"swapped"`
}

// CheckSIMSwap queries the Network as Code API to detect unauthorized SIM card movement.
func (c *Client) CheckSIMSwap(ctx context.Context, msisdn string) (*SIMSwapResult, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	if err := c.rateLimiter.Wait(ctx); err != nil {
		return nil, err
	}

	requestBody, _ := json.Marshal(map[string]interface{}{
		"phoneNumber": msisdn,
		"maxAge":      24, // Check for swaps within the last 24 hours
	})

	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/sim-swap/v0/check", bytes.NewReader(requestBody))

	resp, err := c.DoRequest(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("nokia sim-swap error: %d", resp.StatusCode)
	}

	var result SIMSwapResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result, nil
}

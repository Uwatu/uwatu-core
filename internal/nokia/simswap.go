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

func (c *Client) CheckSIMSwap(ctx context.Context, msisdn string) (*SIMSwapResult, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	c.rateLimiter.Wait(ctx)

	body := []byte(fmt.Sprintf(`{"phoneNumber":"%s", "maxAge": 24}`, msisdn))

	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/sim-swap/v0/check", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result SIMSwapResult
	json.NewDecoder(resp.Body).Decode(&result)
	return &result, nil
}

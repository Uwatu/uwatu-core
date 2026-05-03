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

// QoDSessionResult holds the response from the QoD create session endpoint.
type QoDSessionResult struct {
	SessionID string `json:"sessionId"`
	QosStatus string `json:"qosStatus"`
	StartedAt string `json:"startedAt"`
	ExpiresAt string `json:"expiresAt"`
	Duration  int    `json:"duration"`
}

// CreateQoDSession requests a temporary QoS session for a device.
func (c *Client) CreateQoDSession(ctx context.Context, msisdn, qosProfile string, duration int) (*QoDSessionResult, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	if err := c.rateLimiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limit exceeded: %w", err)
	}

	body := map[string]interface{}{
		"device": map[string]interface{}{
			"phoneNumber": msisdn,
			"ipv4Address": map[string]interface{}{
				"publicAddress":  "233.252.0.2",
				"privateAddress": "192.0.2.25",
				"publicPort":     80,
			},
		},
		"applicationServer": map[string]interface{}{
			"ipv4Address": "8.8.8.8",
		},
		"qosProfile": qosProfile,
		"duration":   duration,
	}

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	endpoint := c.baseURL + "/quality-on-demand/v1/sessions"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	config.LogInfo("HTTP", fmt.Sprintf("--> POST /quality-on-demand/v1/sessions [MSISDN: %s]", msisdn))

	resp, err := c.DoRequest(req)
	if err != nil {
		config.LogError("HTTP", fmt.Sprintf("--> request failed: %v", err))
		return nil, fmt.Errorf("qod request: %w", err)
	}
	defer resp.Body.Close()

	requestID := resp.Header.Get("X-Request-Id")
	// Accept both 200 and 201 (created)
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		bodyBytes, _ := io.ReadAll(resp.Body)
		config.LogError("HTTP", fmt.Sprintf("<-- %d Error [RequestID: %s] | Body: %s", resp.StatusCode, requestID, string(bodyBytes)))
		return nil, fmt.Errorf("qod api error: status %d", resp.StatusCode)
	}

	config.LogInfo("HTTP", fmt.Sprintf("<-- %d OK [RequestID: %s]", resp.StatusCode, requestID))

	var result QoDSessionResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode qod response: %w", err)
	}
	return &result, nil
}

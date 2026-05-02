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

// SliceResult holds the response from the Network Slicing create endpoint.
type SliceResult struct {
	Name  string `json:"name"`
	State string `json:"state"`
	CSIID string `json:"csi_id"`
}

// CreateNetworkSlice requests a dedicated network slice for a device.
// The sandbox returns 202 Accepted while provisioning; the slice ID is the "name" field.
func (c *Client) CreateNetworkSlice(ctx context.Context, msisdn string) (*SliceResult, error) {
	// Give slicing extra time because provisioning is async
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if err := c.rateLimiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limit exceeded: %w", err)
	}

	body := map[string]interface{}{
		"notificationUrl":       "https://example.com/notify",
		"notificationAuthToken": "6207f4c159esfnjsf6280c0e8417090d91710b8c9369f21d9b616ebe809d28c5dc4b1428447fa5d83988b516595253bae24e2a9eb22f43523f2ab1997dd7",
		"networkIdentifier": map[string]interface{}{
			"mcc": "236",
			"mnc": "30",
		},
		"sliceInfo": map[string]interface{}{
			"serviceType":    1,
			"differentiator": "0003E8",
		},
		"maxDataConnections": 42312,
		"maxDevices":         33,
		"sliceUplinkThroughput": map[string]interface{}{
			"guaranteed": 15,
			"maximum":    999999,
		},
		"deviceUplinkThroughput": map[string]interface{}{
			"guaranteed": 10,
			"maximum":    20,
		},
	}

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	endpoint := c.baseURL + "/slice/v1/slices"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	config.LogInfo("HTTP", fmt.Sprintf("--> POST /slice/v1/slices [MSISDN: %s]", msisdn))

	resp, err := c.DoRequest(req)
	if err != nil {
		config.LogError("HTTP", fmt.Sprintf("--> request failed: %v", err))
		return nil, fmt.Errorf("slicing request: %w", err)
	}
	defer resp.Body.Close()

	requestID := resp.Header.Get("X-Request-Id")
	// Accept 200, 201 (created), and 202 (accepted/pending)
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusAccepted {
		bodyBytes, _ := io.ReadAll(resp.Body)
		config.LogError("HTTP", fmt.Sprintf("<-- %d Error [RequestID: %s] | Body: %s", resp.StatusCode, requestID, string(bodyBytes)))
		return nil, fmt.Errorf("slicing api error: status %d", resp.StatusCode)
	}

	config.LogInfo("HTTP", fmt.Sprintf("<-- %d OK [RequestID: %s]", resp.StatusCode, requestID))

	var result SliceResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode slicing response: %w", err)
	}

	config.LogInfo("NOKIA_SLICE", fmt.Sprintf("Slice created: %s (state: %s)", result.Name, result.State))
	return &result, nil
}

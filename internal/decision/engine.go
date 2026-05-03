package decision

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/uwatu/uwatu-core/internal/config"
	"github.com/uwatu/uwatu-core/internal/models"
)

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

var (
	intelURL   = getEnv("INTEL_URL", "http://localhost:8001/score")
	intelToken = getEnv("INTEL_TOKEN", "internal-shared-secret")
)

func CallIntelligence(matrix models.SignalMatrix) models.ScoredEvent {
	body, err := json.Marshal(matrix)
	if err != nil {
		config.LogError("DECISION", "marshal: "+err.Error())
		return safeFallback()
	}

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, intelURL, bytes.NewReader(body))
	if err != nil {
		config.LogError("DECISION", "request: "+err.Error())
		return safeFallback()
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+intelToken)

	resp, err := (&http.Client{Timeout: 500 * time.Millisecond}).Do(req)
	if err != nil {
		config.LogError("DECISION", "call: "+err.Error())
		return safeFallback()
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		config.LogError("DECISION", fmt.Sprintf("status %d", resp.StatusCode))
		return safeFallback()
	}

	var scored models.ScoredEvent
	if err := json.NewDecoder(resp.Body).Decode(&scored); err != nil {
		config.LogError("DECISION", "decode: "+err.Error())
		return safeFallback()
	}

	config.LogInfo("DECISION", fmt.Sprintf("%s (%.0f%%)", scored.EventType, scored.Confidence*100))
	return scored
}

func safeFallback() models.ScoredEvent {
	return models.ScoredEvent{EventType: "NORMAL", Confidence: 0, Suppressed: true}
}

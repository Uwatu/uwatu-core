package alerts_test

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/uwatu/uwatu-core/internal/alerts"
	"github.com/uwatu/uwatu-core/internal/models"
)

func TestAlertRouterWaterfall(t *testing.T) {
	// Pull credentials from the environment
	atAPIKey := os.Getenv("AT_API_KEY")
	atUsername := os.Getenv("AT_SANDBOX_USERNAME")
	atTestPhone := os.Getenv("AT_TEST_PHONE")
	atWhatsAppFrom := os.Getenv("AT_WHATSAPP_NUMBER")
	projectID := os.Getenv("FIREBASE_PROJECT_ID")
	credPath := os.Getenv("FIREBASE_CREDENTIALS")

	if atAPIKey == "" || atUsername == "" || atTestPhone == "" {
		t.Skip("Skipping Router test. Missing basic AT credentials.")
	}

	// Initialize FCM client if possible
	os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", credPath)
	fcmClient, _ := alerts.InitializeFCM(projectID)

	// Initialize the Router (passing nil for DB to bypass Postgres during local testing)
	router := alerts.NewAlertRouter(nil, fcmClient, atAPIKey, atUsername, atWhatsAppFrom)

	dummyToken := "fake-expired-fcm-token-123"

	tests := []struct {
		name     string
		tier     int
		fcmToken *string
	}{
		{
			name:     "Tier 3 Farmer (Triggers FCM -> Falls back to WhatsApp -> SMS)",
			tier:     3,
			fcmToken: &dummyToken,
		},
		{
			name:     "Tier 2 Farmer (Skips FCM -> Tries WhatsApp)",
			tier:     2,
			fcmToken: nil,
		},
		{
			name:     "Tier 0 Farmer (Skips internet -> Direct to SMS)",
			tier:     0,
			fcmToken: nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Build a dummy payload representing the event and the farmer
			payload := models.AlertPayload{
				Farmer: models.Farmer{
					ID:         "farmer-test-123",
					Phone:      atTestPhone,
					DeviceTier: tc.tier,
					FCMToken:   tc.fcmToken,
				},
				Event: models.ScoredEvent{
					EventType: "TEST_BREACH",
				},
				Message: "Uwatu Router Test: Evaluating waterfall for Tier " + string(rune('0'+tc.tier)),
			}

			// Fire the router (this executes in a background goroutine)
			err := router.RouteAlert(payload)
			assert.NoError(t, err)

			// Give the background goroutine 3 seconds to complete its network calls
			time.Sleep(3 * time.Second)
		})
	}
}

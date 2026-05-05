package alerts_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/uwatu/uwatu-core/internal/alerts"
)

func TestFCMIntegration(t *testing.T) {
	// 1. Pull credentials from the environment
	projectID := os.Getenv("FIREBASE_PROJECT_ID")
	credPath := os.Getenv("FIREBASE_CREDENTIALS")

	if projectID == "" || credPath == "" {
		t.Skip("Skipping FCM test. Missing FIREBASE_PROJECT_ID or FIREBASE_CREDENTIALS")
	}

	// 2. Set the GOOGLE_APPLICATION_CREDENTIALS environment variable
	os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", credPath)

	// 3. Test Initialization
	t.Run("Initialize FCM Client", func(t *testing.T) {
		client, err := alerts.InitializeFCM(projectID)
		assert.NoError(t, err, "Should initialize FCM client without errors")
		assert.NotNil(t, client, "Client should not be nil")
	})

	// 4. Test Dispatch (Panic-proofed)
	t.Run("Send Push Notification (Auth Check)", func(t *testing.T) {
		client, err := alerts.InitializeFCM(projectID)
		// Safety check: stop the test gracefully if the client failed to load
		if err != nil {
			t.Fatalf("Failed to initialize client for dispatch test: %v", err)
		}

		dummyToken := "fake-device-token-123456789"
		err = alerts.SendPushNotification(client, dummyToken, "Uwatu Alert", "Test body")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "error sending fcm message", "Should fail due to invalid token, but proves auth works")
	})
}

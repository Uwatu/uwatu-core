package alerts_test

import (
	"io"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/uwatu/uwatu-core/internal/alerts"
)

func TestAfricasTalkingIntegrations(t *testing.T) {
	// 1. Pull credentials from your environment
	apiKey := os.Getenv("AT_API_KEY")
	username := os.Getenv("AT_SANDBOX_USERNAME")
	testPhone := os.Getenv("AT_TEST_PHONE") // The number registered in your AT Simulator

	// --- USSD TEST (No network required) ---
	t.Run("Test USSD Menu", func(t *testing.T) {
		app := fiber.New()
		app.Post("/ussd", alerts.USSDHandler)

		// Simulate an initial USSD dial (empty text)
		req := httptest.NewRequest("POST", "/ussd", strings.NewReader("text="))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)

		body, _ := io.ReadAll(resp.Body)
		assert.Contains(t, string(body), "CON Welcome to Uwatu", "Should return the main menu")
	})

	// --- NETWORK INTEGRATION TESTS ---
	if apiKey == "" || username == "" || testPhone == "" {
		t.Skip("Skipping SMS/WhatsApp tests. Missing AT_API_KEY, AT_SANDBOX_USERNAME, or AT_TEST_PHONE")
	}

	t.Run("Test SMS Alert", func(t *testing.T) {
		err := alerts.SendSMS(apiKey, username, testPhone, "🚨 Uwatu Test: Perimeter breached at Farm Alpha!")
		assert.NoError(t, err, "SMS should send successfully to the AT Sandbox")
	})

	t.Run("Test WhatsApp Alert", func(t *testing.T) {
		if os.Getenv("AT_WHATSAPP_NUMBER") == "" {
			t.Skip("Skipping: WhatsApp sandbox not available (Coming Soon). Set AT_WHATSAPP_NUMBER to test against live.")
		}
		whatsappFrom := os.Getenv("AT_WHATSAPP_NUMBER") // e.g. "+254711082000"
		err := alerts.SendWhatsApp(apiKey, username, whatsappFrom, testPhone, "🚨 Uwatu Test: WhatsApp priority alert triggered!")
		assert.NoError(t, err, "WhatsApp should send successfully")
	})
}

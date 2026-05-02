package alerts

import (
	"github.com/gofiber/fiber/v2"
)

// USSDHandler processes incoming USSD requests from Africa's Talking.
func USSDHandler(c *fiber.Ctx) error {
	text := c.FormValue("text")

	var response string

	switch text {
	case "":
		response = "CON Welcome to Uwatu.\n1. Check Herd Status\n2. Call Vet"
	case "1":
		response = "END Your herd is currently secure. 0 alerts in the last 24h."
	case "2":
		response = "END A local vet has been notified and will SMS you shortly."
	default:
		response = "END Invalid choice. Please try again."
	}

	c.Set("Content-Type", "text/plain")
	return c.SendString(response)
}

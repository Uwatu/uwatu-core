package alerts

import (
	"errors"
	"fmt"
	"log"

	"firebase.google.com/go/v4/messaging"
	"github.com/uwatu/uwatu-core/internal/models"
)

// RouteAlert evaluates the farmer's tier and attempts to deliver the message,
// cascading down to lower-tier methods if network timeouts or failures occur.
func RouteAlert(payload models.AlertPayload, fcmClient *messaging.Client, atAPIKey string, atUsername string, atWhatsAppFrom string) error {
	phone := payload.Farmer.Phone
	message := payload.Message

	switch payload.Farmer.DeviceTier {
	case 3:
		if payload.Farmer.FCMToken != nil {
			token := *payload.Farmer.FCMToken
			err := SendPushNotification(fcmClient, token, payload.Event.EventType, message)

			if err == nil {
				return nil
			}

			if err.Error() == "token_expired" {
				return errors.New("token_expired") // SPECIFIC FAILURE: Tell the main app to delete the token
			}

			// GENERAL FAILURE: Log it, and let it fallthrough to WhatsApp
			log.Printf("FCM failed for %s: %v. Falling back to WhatsApp...", phone, err)
		} else {
			log.Printf("Tier 3 farmer %s has no FCM token. Falling back to WhatsApp...", phone)
		}

		fallthrough

	case 2:
		err := SendWhatsApp(atAPIKey, atUsername, atWhatsAppFrom, phone, message)
		if err == nil {
			return nil
		}

		// FAILURE: Log it, and let it fallthrough to SMS
		log.Printf("WhatsApp failed for %s: %v. Falling back to SMS...", phone, err)

		fallthrough

	case 1:
		err := SendSMS(atAPIKey, atUsername, phone, message)
		if err == nil {
			return nil
		}

		// CRITICAL FAILURE
		return fmt.Errorf("CRITICAL: all alert tiers exhausted for farmer %s. Last error: %v", payload.Farmer.ID, err)

	default:
		return fmt.Errorf("invalid device tier: %d for farmer %s", payload.Farmer.DeviceTier, payload.Farmer.ID)
	}

	return nil
}

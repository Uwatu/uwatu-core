package alerts

import (
	"errors"
	"fmt"
	"log"

	"firebase.google.com/go/v4/messaging"
	"github.com/uwatu/uwatu-core/internal/models"
)

// AlertRouter defines the contract for any router in the system.
// This interface allows us to easily create mock routers for testing.
type AlertRouter interface {
	RouteAlert(payload models.AlertPayload) error
}

// DefaultAlertRouter is the standard implementation of the AlertRouter.
// It acts as a dependency container holding all necessary API clients and credentials.
type DefaultAlertRouter struct {
	FCMClient      *messaging.Client
	ATAPIKey       string
	ATUsername     string
	ATWhatsAppFrom string
}

// NewAlertRouter is the constructor function.
// It runs once during server startup to pack all credentials into the struct.
func NewAlertRouter(fcmClient *messaging.Client, atAPIKey string, atUsername string, atWhatsAppFrom string) AlertRouter {
	return &DefaultAlertRouter{
		FCMClient:      fcmClient,
		ATAPIKey:       atAPIKey,
		ATUsername:     atUsername,
		ATWhatsAppFrom: atWhatsAppFrom,
	}
}

// RouteAlert evaluates the farmer's tier and attempts to deliver the message.
func (r *DefaultAlertRouter) RouteAlert(payload models.AlertPayload) error {
	phone := payload.Farmer.Phone
	message := payload.Message

	switch payload.Farmer.DeviceTier {
	case 3:
		if payload.Farmer.FCMToken != nil {
			token := *payload.Farmer.FCMToken
			err := SendPushNotification(r.FCMClient, token, payload.Event.EventType, message)

			if err == nil {
				return nil
			}

			if err.Error() == "token_expired" {
				return errors.New("token_expired")
			}

			log.Printf("FCM failed for %s: %v. Falling back to WhatsApp...", phone, err)
		} else {
			log.Printf("Tier 3 farmer %s has no FCM token. Falling back to WhatsApp...", phone)
		}

		fallthrough

	case 2:
		err := SendWhatsApp(r.ATAPIKey, r.ATUsername, r.ATWhatsAppFrom, phone, message)
		if err == nil {
			return nil
		}

		log.Printf("WhatsApp failed for %s: %v. Falling back to SMS...", phone, err)

		fallthrough

	case 1:
		err := SendSMS(r.ATAPIKey, r.ATUsername, phone, message)
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

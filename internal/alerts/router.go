package alerts

import (
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
// It fires the routing logic as a background goroutine to unblock the hot path.
func (r *DefaultAlertRouter) RouteAlert(payload models.AlertPayload) error {

	// We pass the payload into the function as 'p' to prevent closure-capture bugs.
	go func(p models.AlertPayload) {
		phone := p.Farmer.Phone
		message := p.Message

		switch p.Farmer.DeviceTier {
		case 3:
			if p.Farmer.FCMToken != nil {
				token := *p.Farmer.FCMToken
				err := SendPushNotification(r.FCMClient, token, p.Event.EventType, message)
				if err == nil {
					return
				}
				if err.Error() == "token_expired" {
					log.Println("Token expired")
					return
				}
				log.Printf("FCM failed for %s: %v. Falling back to WhatsApp...", phone, err)
			} else {
				log.Printf("Tier 3 farmer %s has no FCM token. Falling back to WhatsApp...", phone)
			}
			fallthrough

		case 2:
			err := SendWhatsApp(r.ATAPIKey, r.ATUsername, r.ATWhatsAppFrom, phone, message)
			if err == nil {
				return
			}
			log.Printf("WhatsApp failed for %s: %v. Falling back to SMS...", phone, err)
			fallthrough

		case 1:
			// We skip USSD push because async network-initiated USSD is unreliable/unsupported.
			log.Printf("Skipping USSD push for Tier 1 farmer %s. Falling back directly to SMS...", phone)
			fallthrough

		case 0:
			err := SendSMS(r.ATAPIKey, r.ATUsername, phone, message)
			if err == nil {
				return
			}
			log.Printf("CRITICAL: all alert tiers exhausted for farmer %s. Last error: %v", p.Farmer.ID, err)
			return
		default:
			log.Printf("invalid device tier: %d for farmer %s", p.Farmer.DeviceTier, p.Farmer.ID)
			return
		}

	}(payload)

	return nil
}

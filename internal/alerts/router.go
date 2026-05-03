package alerts

import (
	"context"
	"log"

	"firebase.google.com/go/v4/messaging"
	"github.com/jackc/pgx/v5/pgxpool"
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
	DB             *pgxpool.Pool
	FCMClient      *messaging.Client
	ATAPIKey       string
	ATUsername     string
	ATWhatsAppFrom string
}

// NewAlertRouter is the constructor function.
// It runs once during server startup to pack all credentials into the struct.
func NewAlertRouter(db *pgxpool.Pool, fcmClient *messaging.Client, atAPIKey string, atUsername string, atWhatsAppFrom string) AlertRouter {
	return &DefaultAlertRouter{
		DB:             db,
		FCMClient:      fcmClient,
		ATAPIKey:       atAPIKey,
		ATUsername:     atUsername,
		ATWhatsAppFrom: atWhatsAppFrom,
	}
}

// RouteAlert evaluates the farmer's tier and attempts to deliver the message.
// It fires the routing logic as a background goroutine to unblock the hot path.
func (r *DefaultAlertRouter) RouteAlert(payload models.AlertPayload) error {

	go func(p models.AlertPayload) {
		phone := p.Farmer.Phone
		message := p.Message
		farmerID := p.Farmer.ID

		switch p.Farmer.DeviceTier {
		case 3:
			if p.Farmer.FCMToken != nil {
				token := *p.Farmer.FCMToken
				err := SendPushNotification(r.FCMClient, token, p.Event.EventType, message)

				if err == nil {
					r.logToDB(farmerID, "FCM", "sent", "")
					return
				}

				r.logToDB(farmerID, "FCM", "failed", err.Error())

				if err.Error() == "token_expired" {
					log.Println("Token expired")
					return
				}
				log.Printf("FCM failed for %s: %v. Falling back to WhatsApp...", phone, err)
			} else {
				r.logToDB(farmerID, "FCM", "failed", "no_token")
				log.Printf("Tier 3 farmer %s has no FCM token. Falling back to WhatsApp...", phone)
			}
			fallthrough

		case 2:
			err := SendWhatsApp(r.ATAPIKey, r.ATUsername, r.ATWhatsAppFrom, phone, message)
			if err == nil {
				r.logToDB(farmerID, "WhatsApp", "sent", "")
				return
			}

			r.logToDB(farmerID, "WhatsApp", "failed", err.Error())
			log.Printf("WhatsApp failed for %s: %v. Falling back to SMS...", phone, err)
			fallthrough

		case 1:
			// We skip USSD push because async network-initiated USSD is unreliable/unsupported.
			log.Printf("Skipping USSD push for Tier 1 farmer %s. Falling back directly to SMS...", phone)
			fallthrough

		case 0:
			err := SendSMS(r.ATAPIKey, r.ATUsername, phone, message)
			if err == nil {
				r.logToDB(farmerID, "SMS", "sent", "")
				return
			}

			r.logToDB(farmerID, "SMS", "failed", err.Error())
			log.Printf("CRITICAL: all alert tiers exhausted for farmer %s. Last error: %v", farmerID, err)
			return

		default:
			log.Printf("invalid device tier: %d for farmer %s", p.Farmer.DeviceTier, farmerID)
			return
		}

	}(payload)

	return nil
}

// logToDB writes the outcome of a notification attempt to the Postgres database.
func (r *DefaultAlertRouter) logToDB(farmerID string, channel string, status string, errorMsg string) {
	// If the DB isn't connected (e.g., during local testing), just skip.
	if r.DB == nil {
		return
	}

	// We use a background context because we don't want DB latency to slow down the router thread
	query := `INSERT INTO notification_log (farmer_id, channel, status, error_msg, sent_at) VALUES ($1, $2, $3, $4, NOW())`

	// Execute the query using pgxpool
	_, err := r.DB.Exec(context.Background(), query, farmerID, channel, status, errorMsg)
	if err != nil {
		log.Printf("Failed to write to notification_log: %v", err)
	}
}

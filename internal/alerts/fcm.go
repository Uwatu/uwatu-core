package alerts

import (
	"context"
	"fmt"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
)

// InitializeFCM connects to Google's servers and returns a ready-to-use messaging client.
func InitializeFCM(projectID string) (*messaging.Client, error) {
	// Create a background context for the initialization
	ctx := context.Background()

	// Set up the Firebase configuration using the projectID
	conf := &firebase.Config{ProjectID: projectID}

	firebaseInstance, err := firebase.NewApp(ctx, conf)
	if err != nil {
		return nil, fmt.Errorf("error initializing firebase app: %v", err)
	}

	messagingClient, err := firebaseInstance.Messaging(ctx)

	if err != nil {
		return nil, fmt.Errorf("error initializing messaging client: %v", err)
	}
	return messagingClient, nil
}

// SendPushNotification sends a rich push alert to a specific smartphone.
func SendPushNotification(client *messaging.Client, fcmToken string, title string, body string) error {
	// A Context controls the timeout and cancellation of the request
	ctx := context.Background()

	message := &messaging.Message{
		Token:        fcmToken,
		Notification: &messaging.Notification{Title: title, Body: body},
	}

	_, err := client.Send(ctx, message)
	if err != nil {
		return fmt.Errorf("error sending fcm message: %v", err)
	}

	return nil
}

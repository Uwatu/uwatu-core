package nokia

import (
	"context"
	"net/http"
	"time"

	"golang.org/x/oauth2/clientcredentials"
	"golang.org/x/time/rate"
)

type Client struct {
	httpClient  *http.Client
	baseURL     string
	rateLimiter *rate.Limiter
}

// NewClient sets up the auto-login to Nokia
func NewClient(clientID, clientSecret, baseURL string) *Client {
	cfg := &clientcredentials.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		TokenURL:     baseURL + "/oauth2/token",
		Scopes: []string{
			"dpv:FraudPreventionAndDetection",
			"device-location:read",
			// 2 of the 9 for now
		},
	}

	return &Client{

		// cfg.Client() automatically gets and attaches your VIP pass to all requests

		httpClient: cfg.Client(context.Background()),
		baseURL:    baseURL,
		// Limit to 100 requests per minute so we don't spam nokia
		rateLimiter: rate.NewLimiter(rate.Every(time.Minute/100), 1),
	}
}

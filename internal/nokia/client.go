package nokia

import (
	_ "context"
	"net/http"
	"time"

	"golang.org/x/time/rate"
)

type Client struct {
	httpClient  *http.Client
	baseURL     string
	apiKey      string
	apiHost     string
	rateLimiter *rate.Limiter // To keep us under 100 req/min
}

func NewClient(apiKey, apiHost, baseURL string) *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 10 * time.Second},
		baseURL:    baseURL,
		apiKey:     apiKey,
		apiHost:    apiHost,
		// Limit to 1 request every 600ms (roughly 100 per minute)
		rateLimiter: rate.NewLimiter(rate.Every(time.Minute/100), 8),
	}

}

// DoRequest is a helper that adds the RapidAPI headers automatically
func (c *Client) DoRequest(req *http.Request) (*http.Response, error) {
	req.Header.Set("x-rapidapi-key", c.apiKey)
	req.Header.Set("x-rapidapi-host", c.apiHost)
	req.Header.Set("Content-Type", "application/json")

	return c.httpClient.Do(req)
}

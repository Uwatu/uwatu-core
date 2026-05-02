package alerts

import (
	"net/http"
	"time"
)

// AlertHTTPClient is a shared, thread-safe HTTP client with a strict timeout.
// It is used by all dispatchers (SMS, WhatsApp) to prevent hanging requests.
var AlertHTTPClient = &http.Client{
	Timeout: time.Duration(8 * time.Second),
}

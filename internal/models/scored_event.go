package models

// ScoredEvent is the response from the Python intelligence layer.
type ScoredEvent struct {
	EventType       string   `json:"event_type"`
	Confidence      float64  `json:"confidence"`
	Suppressed      bool     `json:"suppressed"`
	AlertChannels   []string `json:"alert_channels"`
	GeminiNarrative string   `json:"gemini_narrative"`
}

// AlertPayload is the final struct handed to the dispatchers.
type AlertPayload struct {
	Event   ScoredEvent
	Farmer  Farmer
	Message string
}

package models

type NokiaSignals struct {
	Lat             float64 `json:"lat"`
	Lon             float64 `json:"lon"`
	SimSwapped      bool    `json:"sim_swapped"`
	DeviceReachable string  `json:"device_reachable"`
}

type Baseline struct {
	AvgTemp7d    float64 `json:"avg_temp_7d"`
	AvgAccel7d   float64 `json:"avg_accel_7d"`
	LocationRisk float64 `json:"location_risk_score"`
}

type EnvironmentContext struct {
	IsNight                       bool `json:"is_night"`
	IsDrySeason                   bool `json:"is_dry_season"`
	MarketDay                     bool `json:"market_day"`
	MinutesSinceGeofenceDeparture *int `json:"minutes_since_geofence_departure"`
}

// SignalMatrix is the final enriched payload sent to uwatu-intelligence.
type SignalMatrix struct {
	DeviceID string `json:"device_id"`
	MSISDN   string `json:"msisdn"`
	FarmID   string `json:"farm_id"`
	AnimalID string `json:"animal_id"`

	Telemetry TagTelemetry       `json:"firmware_payload"` // Maps to simulator's object name
	Nokia     NokiaSignals       `json:"nokia_signals"`
	Baseline  Baseline           `json:"baseline"`
	Context   EnvironmentContext `json:"context"`
}

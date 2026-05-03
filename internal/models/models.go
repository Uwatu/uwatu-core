package models

import "time"

type Point struct {
	Lat float64 `json:"latitude"`
	Lon float64 `json:"longitude"`
}

// Polygon represents a geofence boundary in memory.
type Polygon []Point

type Farmer struct {
	ID         string
	Name       string
	Phone      string
	DeviceTier int
	Locale     string
	FCMToken   *string
}

type Farm struct {
	ID             string
	FarmerID       string
	Name           string
	Geofence       Polygon
	DrySeasonStart time.Month
	DrySeasonEnd   time.Month
}

type Animal struct {
	ID      string
	FarmID  string
	Species string
}

type Tag struct {
	DeviceID string
	AnimalID string
}

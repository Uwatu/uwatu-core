package ws

import (
	"context"
	"fmt"
	"log"

	"github.com/uwatu/uwatu-core/internal/farm"
	"github.com/uwatu/uwatu-core/internal/geofence"
	"github.com/uwatu/uwatu-core/internal/models"
)

// LocationPayload represents the raw data coming from the tracking collar.
type LocationPayload struct {
	DeviceID string  `json:"device_id"`
	Lat      float64 `json:"lat"`
	Lon      float64 `json:"lon"`
}

// Tracker handles processing incoming location events.
type Tracker struct {
	animals   *farm.AnimalRegistry
	geofences *geofence.Manager
}

// NewTracker creates a new instance of the tracking pipeline.
func NewTracker(a *farm.AnimalRegistry, g *geofence.Manager) *Tracker {
	return &Tracker{
		animals:   a,
		geofences: g,
	}
}

// ProcessLocationUpdate runs the core tracking pipeline every time a ping arrives.
func (t *Tracker) ProcessLocationUpdate(ctx context.Context, payload LocationPayload) error {

	animal, err := t.animals.GetAnimalByDeviceID(ctx, payload.DeviceID)

	if err != nil {
		return fmt.Errorf("Could not get animal from animals registry: %w", err)
	}

	polygon, ok := t.geofences.Farms[animal.FarmID]

	if !ok {
		return fmt.Errorf("Farm boundary isn't loaded")
	}

	point := models.Point{
		Lat: payload.Lat,
		Lon: payload.Lon,
	}

	isInside := geofence.IsInside(point, polygon)

	if !isInside {
		log.Printf("ALERT: Animal %s has left Farm %s!", animal.ID, animal.FarmID)
	}

	return nil
}

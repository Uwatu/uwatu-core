package ingestion

import (
	"context"
	"fmt"
	"sync"

	"github.com/uwatu/uwatu-core/internal/config"
	"github.com/uwatu/uwatu-core/internal/models" // <-- Check your go.mod for the real path!
	"github.com/uwatu/uwatu-core/internal/nokia"
)

type Enricher struct {
	nokiaClient *nokia.Client
}

func NewEnricher(nc *nokia.Client) *Enricher {
	return &Enricher{nokiaClient: nc}
}

// Process fires the Nokia APIs and builds the final Matrix
func (e *Enricher) Process(deviceID string, msisdn string, battery int) {
	ctx := context.Background()

	// 1. Start building the package
	matrix := models.SignalMatrix{
		DeviceID:   deviceID,
		MSISDN:     msisdn,
		BatteryPct: battery,
	}

	// 2. We use a WaitGroup to do 2 Nokia calls at the exact same time
	var wg sync.WaitGroup
	wg.Add(2)

	// Background Task A: Get Location
	go func() {
		defer wg.Done()
		loc, err := e.nokiaClient.GetDeviceLocation(ctx, msisdn)
		if err == nil && loc != nil {
			matrix.Lat = loc.Lat
			matrix.Lon = loc.Lon
		}
	}()

	// Background Task B: Check SIM Swap
	go func() {
		defer wg.Done()
		swap, err := e.nokiaClient.CheckSIMSwap(ctx, msisdn)
		if err == nil && swap != nil {
			matrix.SimSwapped = swap.Swapped
		}
	}()

	// 3. Wait for both Nokia calls to finish
	wg.Wait()

	// 4. Print the result!
	summary := fmt.Sprintf("ID: %s | Lat: %.4f | Stolen: %v", matrix.DeviceID, matrix.Lat, matrix.SimSwapped)
	config.LogSuccess("ENRICH", summary)

	// mpheles alert takes over
}

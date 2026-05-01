package ingestion

import (
	"context"
	"sync"
	"time"

	"github.com/uwatu/uwatu-core/internal/config"
	"github.com/uwatu/uwatu-core/internal/models"
	"github.com/uwatu/uwatu-core/internal/nokia"
)

// networkState stores the last retrieved data from Nokia to prevent
// unnecessary API consumption while maintaining data continuity.
type networkState struct {
	lat        float64
	lon        float64
	simSwapped bool
	lastFetch  time.Time
}

type Enricher struct {
	nokiaClient *nokia.Client
	mu          sync.RWMutex
	cache       map[string]*networkState
}

func NewEnricher(nc *nokia.Client) *Enricher {
	return &Enricher{
		nokiaClient: nc,
		cache:       make(map[string]*networkState),
	}
}

// Process coordinates the fusion of real-time sensor data with periodic network-layer signals.
func (e *Enricher) Process(deviceID string, msisdn string, battery int, temp float64, accel int) {
	// Initialize the signal matrix with real-time firmware data
	matrix := models.SignalMatrix{
		DeviceID:   deviceID,
		MSISDN:     msisdn,
		BatteryPct: battery,
		Temp:       temp,
		Accel:      accel,
	}

	// 1. Thread-safe cache lookup
	e.mu.RLock()
	state, exists := e.cache[deviceID]
	e.mu.RUnlock()

	// 2. Determine if network-layer refresh is required (2-minute cadence)
	if !exists || time.Since(state.lastFetch) > (2*time.Minute) {
		e.refreshNetworkSignals(context.Background(), &matrix)

		// 3. Update cache with fresh network results
		e.mu.Lock()
		e.cache[deviceID] = &networkState{
			lat:        matrix.Lat,
			lon:        matrix.Lon,
			simSwapped: matrix.SimSwapped,
			lastFetch:  time.Now(),
		}
		e.mu.Unlock()
	} else {
		// 4. Use cached network data for continuity
		matrix.Lat = state.lat
		matrix.Lon = state.lon
		matrix.SimSwapped = state.simSwapped
	}

	e.logTelemetry(&matrix)
}

// refreshNetworkSignals executes parallel calls to Nokia APIs to minimize total latency.
func (e *Enricher) refreshNetworkSignals(ctx context.Context, matrix *models.SignalMatrix) {
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		if loc, err := e.nokiaClient.GetDeviceLocation(ctx, matrix.MSISDN); err == nil {
			matrix.Lat = loc.Area.Center.Lat
			matrix.Lon = loc.Area.Center.Lon
		}
	}()

	go func() {
		defer wg.Done()
		if swap, err := e.nokiaClient.CheckSIMSwap(ctx, matrix.MSISDN); err == nil {
			matrix.SimSwapped = swap.Swapped
		}
	}()

	wg.Wait()
}

func (e *Enricher) logTelemetry(m *models.SignalMatrix) {
	config.LogEnrich(m.DeviceID, m.Temp, m.Accel, m.BatteryPct, m.Lat, m.Lon, m.SimSwapped)
}

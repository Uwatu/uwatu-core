package ingestion

import (
	"context"
	"sync"
	"time"

	"github.com/uwatu/uwatu-core/internal/config"
	"github.com/uwatu/uwatu-core/internal/models"
	"github.com/uwatu/uwatu-core/internal/nokia"
)

// networkState stores the last retrieved Nokia data for a device.
type networkState struct {
	lat        float64
	lon        float64
	simSwapped bool
	lastFetch  time.Time
}

// Enricher coordinates the fusion of real-time firmware telemetry with
// Nokia network-layer signals, using a two‑minute cache to limit API calls.
type Enricher struct {
	nokiaClient *nokia.Client
	mu          sync.RWMutex
	cache       map[string]*networkState
}

// NewEnricher creates a new Enricher with an empty cache.
func NewEnricher(nc *nokia.Client) *Enricher {
	return &Enricher{
		nokiaClient: nc,
		cache:       make(map[string]*networkState),
	}
}

// Process accepts a device ID, MSISDN and the already‑parsed firmware telemetry,
// constructs a SignalMatrix, refreshes Nokia data if the cache is stale, and
// logs the enriched telemetry to the terminal.
func (e *Enricher) Process(deviceID, msisdn string, telemetry models.TagTelemetry) {
	matrix := models.SignalMatrix{
		DeviceID:  deviceID,
		MSISDN:    msisdn,
		Telemetry: telemetry,
		// FarmID, AnimalID, Baseline and Context will be populated
		// later by the farm registry and intelligence layer.
	}

	// 1. Thread‑safe read from cache
	e.mu.RLock()
	state, exists := e.cache[deviceID]
	e.mu.RUnlock()

	// 2. Refresh network signals if needed (every 2 minutes)
	if !exists || time.Since(state.lastFetch) > 2*time.Minute {
		e.refreshNetworkSignals(context.Background(), &matrix)

		// 3. Update cache with fresh values (or zero if calls failed)
		e.mu.Lock()
		e.cache[deviceID] = &networkState{
			lat:        matrix.Nokia.Lat,
			lon:        matrix.Nokia.Lon,
			simSwapped: matrix.Nokia.SimSwapped,
			lastFetch:  time.Now(),
		}
		e.mu.Unlock()
	} else {
		// Use cached network state
		matrix.Nokia.Lat = state.lat
		matrix.Nokia.Lon = state.lon
		matrix.Nokia.SimSwapped = state.simSwapped
	}

	e.logTelemetry(&matrix)
}

// refreshNetworkSignals calls the Nokia APIs in parallel and populates
// the matrix's NokiaSignals field. Failures are logged; the pipeline
// continues with whatever was returned (zero values on error).
func (e *Enricher) refreshNetworkSignals(ctx context.Context, matrix *models.SignalMatrix) {
	var wg sync.WaitGroup
	wg.Add(4) // doing 4 api calls (swap, location, device status, roaming)

	// LOCATION API
	go func() {
		defer wg.Done()
		loc, err := e.nokiaClient.GetDeviceLocation(ctx, matrix.MSISDN)
		if err != nil {
			config.LogError("NOKIA_LOC", err.Error())
			return
		}
		matrix.Nokia.Lat = loc.Area.Center.Lat
		matrix.Nokia.Lon = loc.Area.Center.Lon
	}()

	// SIM SWAP API
	go func() {
		defer wg.Done()
		swap, err := e.nokiaClient.CheckSIMSwap(ctx, matrix.MSISDN)
		if err != nil {
			config.LogError("NOKIA_SWAP", err.Error())
			return
		}
		matrix.Nokia.SimSwapped = swap.Swapped
	}()

	// DEVICE STATUS API
	go func() {
		defer wg.Done()
		status, err := e.nokiaClient.GetDeviceStatus(ctx, matrix.MSISDN)
		if err != nil {
			config.LogError("NOKIA_STATUS", err.Error())
			return
		}
		// Map reachability to a descriptive string
		if status.Reachable {
			if len(status.Connectivity) > 0 && status.Connectivity[0] == "SMS" {
				matrix.Nokia.DeviceReachable = "REACHABLE_SMS"
			} else {
				matrix.Nokia.DeviceReachable = "REACHABLE_DATA"
			}
		} else {
			matrix.Nokia.DeviceReachable = "UNREACHABLE"
		}
	}()

	// extra apis (for as much data as possible)

	// roaming
	go func() {
		defer wg.Done()
		roam, err := e.nokiaClient.GetRoamingStatus(ctx, matrix.MSISDN)
		if err != nil {
			config.LogError("NOKIA_ROAM", err.Error())
			return
		}
		matrix.Nokia.Roaming = roam.Roaming
		matrix.Nokia.RoamingCountry = roam.CountryCode
	}()

	wg.Wait()
}

// logTelemetry prints the enriched telemetry line using the ANSI logger.
func (e *Enricher) logTelemetry(m *models.SignalMatrix) {
	config.LogEnrich(
		m.DeviceID,
		m.Telemetry.BodyTempC,
		m.Telemetry.AccelMagnitude,
		m.Telemetry.BatteryPct,
		m.Nokia.Lat,
		m.Nokia.Lon,
		m.Nokia.SimSwapped,
	)
}

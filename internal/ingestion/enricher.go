package ingestion

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/uwatu/uwatu-core/internal/config"
	"github.com/uwatu/uwatu-core/internal/models"
	"github.com/uwatu/uwatu-core/internal/nokia"
)

// networkState stores the last retrieved Nokia data for a device.
type networkState struct {
	lat             float64
	lon             float64
	simSwapped      bool
	deviceSwapped   bool
	roaming         bool
	roamingCountry  int
	deviceReachable string
	congestionLevel string
	lastFetch       time.Time
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
	}

	// 1. Thread‑safe read from cache
	e.mu.RLock()
	state, exists := e.cache[deviceID]
	e.mu.RUnlock()

	// 2. Refresh network signals if needed (every 2 minutes)
	if !exists || time.Since(state.lastFetch) > 2*time.Minute {
		e.refreshNetworkSignals(context.Background(), &matrix)

		// 3. Update cache with fresh values
		e.mu.Lock()
		e.cache[deviceID] = &networkState{
			lat:             matrix.Nokia.Lat,
			lon:             matrix.Nokia.Lon,
			simSwapped:      matrix.Nokia.SimSwapped,
			deviceSwapped:   matrix.Nokia.DeviceSwapped,
			roaming:         matrix.Nokia.Roaming,
			roamingCountry:  matrix.Nokia.RoamingCountry,
			deviceReachable: matrix.Nokia.DeviceReachable,
			congestionLevel: matrix.Nokia.CongestionLevel,
			lastFetch:       time.Now(),
		}
		e.mu.Unlock()
	} else {
		// Use cached network state
		matrix.Nokia.Lat = state.lat
		matrix.Nokia.Lon = state.lon
		matrix.Nokia.SimSwapped = state.simSwapped
		matrix.Nokia.DeviceSwapped = state.deviceSwapped
		matrix.Nokia.Roaming = state.roaming
		matrix.Nokia.RoamingCountry = state.roamingCountry
		matrix.Nokia.DeviceReachable = state.deviceReachable
		matrix.Nokia.CongestionLevel = state.congestionLevel
	}

	e.logTelemetry(&matrix)
}

// refreshNetworkSignals calls the Nokia APIs in parallel and populates
// the matrix's NokiaSignals field. Failures are logged; the pipeline
// continues with whatever was returned (zero values on error).
func (e *Enricher) refreshNetworkSignals(ctx context.Context, matrix *models.SignalMatrix) {
	var wg sync.WaitGroup
	wg.Add(5)

	delays := []time.Duration{0, 150 * time.Millisecond, 300 * time.Millisecond, 450 * time.Millisecond, 600 * time.Millisecond}

	// 1. Location
	go func() {
		defer wg.Done()
		time.Sleep(delays[0])
		loc, err := e.nokiaClient.GetDeviceLocation(ctx, matrix.MSISDN)
		if err != nil {
			config.LogError("NOKIA_LOC", err.Error())
			return
		}
		matrix.Nokia.Lat = loc.Area.Center.Lat
		matrix.Nokia.Lon = loc.Area.Center.Lon
	}()

	// 2. SIM Swap
	go func() {
		defer wg.Done()
		time.Sleep(delays[1])
		swap, err := e.nokiaClient.CheckSIMSwap(ctx, matrix.MSISDN)
		if err != nil {
			config.LogError("NOKIA_SWAP", err.Error())
			return
		}
		matrix.Nokia.SimSwapped = swap.Swapped
	}()

	// 3. Device Reachability
	go func() {
		defer wg.Done()
		time.Sleep(delays[2])
		status, err := e.nokiaClient.GetDeviceStatus(ctx, matrix.MSISDN)
		if err != nil {
			config.LogError("NOKIA_STATUS", err.Error())
			return
		}
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

	// 4. Roaming Status
	go func() {
		defer wg.Done()
		time.Sleep(delays[3])
		roam, err := e.nokiaClient.GetRoamingStatus(ctx, matrix.MSISDN)
		if err != nil {
			config.LogError("NOKIA_ROAM", err.Error())
			return
		}
		matrix.Nokia.Roaming = roam.Roaming
		matrix.Nokia.RoamingCountry = roam.CountryCode
		config.LogInfo("NOKIA_ROAM", fmt.Sprintf("Roaming: %t | Country: %d", roam.Roaming, roam.CountryCode))
	}()

	// 5. Device Swap
	go func() {
		defer wg.Done()
		time.Sleep(delays[4])
		devSwap, err := e.nokiaClient.CheckDeviceSwap(ctx, matrix.MSISDN, 120)
		if err != nil {
			config.LogError("NOKIA_DEVSWAP", err.Error())
			return
		}
		matrix.Nokia.DeviceSwapped = devSwap.Swapped
	}()

	wg.Wait()

	// Stub Congestion Insights until a webhook receiver is implemented.
	// Real integration will push updates; for now we assume normal conditions.
	matrix.Nokia.CongestionLevel = "Low"
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
		m.Nokia.DeviceSwapped,
		m.Nokia.Roaming,
		m.Nokia.DeviceReachable,
		m.Nokia.CongestionLevel, // will be empty until real integration
	)
}

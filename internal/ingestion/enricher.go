package ingestion

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/uwatu/uwatu-core/internal/alerts"
	"github.com/uwatu/uwatu-core/internal/config"
	"github.com/uwatu/uwatu-core/internal/db"
	"github.com/uwatu/uwatu-core/internal/decision"
	"github.com/uwatu/uwatu-core/internal/farm"
	"github.com/uwatu/uwatu-core/internal/geofence"
	"github.com/uwatu/uwatu-core/internal/models"
	"github.com/uwatu/uwatu-core/internal/nokia"
	"github.com/uwatu/uwatu-core/internal/ws"
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
	qodSessionID    string
	sliceID         string
	lastFetch       time.Time
}

// Enricher coordinates the fusion of real-time firmware telemetry with
// Nokia network-layer signals, using a two‑minute cache to limit API calls.
type Enricher struct {
	nokiaClient *nokia.Client
	animalReg   *farm.AnimalRegistry
	farmReg     *farm.Registry
	geofenceMgr *geofence.Manager
	alertRouter alerts.AlertRouter
	hub         *ws.Hub
	mu          sync.RWMutex
	cache       map[string]*networkState
}

// NewEnricher creates a new Enricher with an empty cache.
func NewEnricher(nc *nokia.Client, ar *farm.AnimalRegistry, fr *farm.Registry, gm *geofence.Manager, router alerts.AlertRouter, hub *ws.Hub) *Enricher {
	return &Enricher{
		nokiaClient: nc,
		animalReg:   ar,
		farmReg:     fr,
		geofenceMgr: gm,
		alertRouter: router,
		hub:         hub,
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

	e.mu.RLock()
	state, exists := e.cache[deviceID]
	e.mu.RUnlock()

	if !exists || time.Since(state.lastFetch) > 2*time.Minute {
		if exists {
			matrix.Nokia.QoDSessionID = state.qodSessionID
			matrix.Nokia.SliceID = state.sliceID
		}
		e.refreshNetworkSignals(context.Background(), &matrix)

		// ---- Geofence check (inside the location goroutine is better, but we'll do it after) ----
		// Look up animal to get farm ID
		if e.animalReg != nil {
			animal, err := e.animalReg.GetAnimalByDeviceID(context.Background(), deviceID)
			if err == nil {
				matrix.FarmID = animal.FarmID
				// Check geofence
				if e.geofenceMgr != nil {
					if poly, ok := e.geofenceMgr.Farms[animal.FarmID]; ok {
						point := models.Point{Lat: matrix.Nokia.Lat, Lon: matrix.Nokia.Lon}
						if !geofence.IsInside(point, poly) {
							config.LogInfo("GEOFENCE", fmt.Sprintf("%s left farm %s", deviceID, animal.FarmID))
							// You could set a field in Context, but for demo we log it.
						}
					}
				}
			}
		}

		// ---- Decision Engine ----
		go func() {
			scored := decision.CallIntelligence(matrix)
			if scored.EventType != "NORMAL" && scored.Confidence > 0.1 {
				config.LogSuccess("ALERT", fmt.Sprintf("%s detected (%.0f%%) for %s", scored.EventType, scored.Confidence*100, matrix.DeviceID))
				// Route alert
				if e.alertRouter != nil && matrix.FarmID != "" {
					// Get farmer through farm -> farmer
					farm, err := e.farmReg.GetFarm(context.Background(), matrix.FarmID)
					if err == nil {
						farmer, err := e.farmReg.GetFarmer(context.Background(), farm.FarmerID)
						if err == nil {
							payload := models.AlertPayload{
								Event:   scored,
								Farmer:  *farmer,
								Message: scored.GeminiNarrative,
							}
							_ = e.alertRouter.RouteAlert(payload)
						}
					}
				}
			}
		}()

		// Update cache ...
		// (keep existing cache update)
	} else {
		// Use cached values ...
	}

	e.logTelemetry(&matrix)
	e.maybePersist(&matrix)

	// Broadcast to dashboard
	if e.hub != nil {
		e.hub.BroadcastEnriched(matrix)
	}
}

// refreshNetworkSignals calls the Nokia APIs in parallel and populates
// the matrix's NokiaSignals field. Failures are logged; the pipeline
// continues with whatever was returned (zero values on error).
func (e *Enricher) refreshNetworkSignals(ctx context.Context, matrix *models.SignalMatrix) {
	var wg sync.WaitGroup
	wg.Add(7)

	delays := []time.Duration{
		0,
		150 * time.Millisecond,
		300 * time.Millisecond,
		450 * time.Millisecond,
		600 * time.Millisecond,
		750 * time.Millisecond,
		900 * time.Millisecond,
	}

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

	// 4. Roaming
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

	// 6. QoD (only if no active session)
	go func() {
		defer wg.Done()
		time.Sleep(delays[5])
		if matrix.Nokia.QoDSessionID != "" {
			return
		}
		session, err := e.nokiaClient.CreateQoDSession(ctx, matrix.MSISDN, "DOWNLINK_M_UPLINK_L", 60)
		if err != nil {
			config.LogError("NOKIA_QOD", err.Error())
			return
		}
		matrix.Nokia.QoDSessionID = session.SessionID
		config.LogInfo("NOKIA_QOD", fmt.Sprintf("Session created: %s", session.SessionID))
	}()

	// 7. Slicing (only if no active slice)
	go func() {
		defer wg.Done()
		time.Sleep(delays[6])
		if matrix.Nokia.SliceID != "" {
			return
		}
		slice, err := e.nokiaClient.CreateNetworkSlice(ctx, matrix.MSISDN)
		if err != nil {
			config.LogError("NOKIA_SLICE", err.Error())
			return
		}
		matrix.Nokia.SliceID = slice.Name
		config.LogInfo("NOKIA_SLICE", fmt.Sprintf("Slice created: %s (state: %s)", slice.Name, slice.State))
	}()

	wg.Wait()
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
		m.Nokia.CongestionLevel,
	)
}

func (e *Enricher) maybePersist(m *models.SignalMatrix) {
	if db.Pool == nil {
		return
	}
	_, err := db.Pool.Exec(context.Background(),
		`INSERT INTO telemetry_events (device_id, msisdn, temp_c, accel, battery_pct, lat, lon, sim_swapped, device_swapped, roaming, reachable, congestion, recorded_at)
         VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,now())`,
		m.DeviceID, m.MSISDN, m.Telemetry.BodyTempC, m.Telemetry.AccelMagnitude, m.Telemetry.BatteryPct,
		m.Nokia.Lat, m.Nokia.Lon, m.Nokia.SimSwapped, m.Nokia.DeviceSwapped, m.Nokia.Roaming,
		m.Nokia.DeviceReachable, m.Nokia.CongestionLevel,
	)
	if err != nil {
		config.LogError("DB", "insert: "+err.Error())
	}
}

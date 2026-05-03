package geofence

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/uwatu/uwatu-core/internal/models"
)

// Manager holds all farm boundaries in memory for zero-latency lookups.
type Manager struct {
	db    *pgxpool.Pool
	mu    sync.RWMutex
	Farms map[string]models.Polygon // Key: FarmID, Value: The Polygon
}

func NewManager(db *pgxpool.Pool) *Manager {
	return &Manager{
		db:    db,
		Farms: make(map[string]models.Polygon),
	}
}

// LoadAll queries the database and populates the in-memory map.
func (m *Manager) LoadAll(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	query := `
		SELECT id, ST_AsGeoJSON(boundary)
		FROM farms
		WHERE boundary IS NOT NULL
	`

	rows, err := m.db.Query(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to query geofences: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var farmID string
		var geoJSONStr string

		err := rows.Scan(&farmID, &geoJSONStr)
		if err != nil {
			return fmt.Errorf("failed to query geofence: %w", err)
		}

		// Parse the GeoJSON string
		var geo struct {
			Coordinates [][][]float64 `json:"coordinates"`
		}
		if err := json.Unmarshal([]byte(geoJSONStr), &geo); err != nil {
			continue // Skip broken polygons
		}

		var poly models.Polygon
		if len(geo.Coordinates) > 0 && len(geo.Coordinates[0]) > 0 {
			for _, coord := range geo.Coordinates[0] {
				point := models.Point{
					Lat: coord[1],
					Lon: coord[0],
				}
				poly = append(poly, point)
			}
		}

		m.Farms[farmID] = poly
	}

	return nil
}

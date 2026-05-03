package farm

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/uwatu/uwatu-core/internal/models"
)

// Registry handles all database operations for Farms and Farmers.
type Registry struct {
	db *pgxpool.Pool
}

// NewRegistry creates a new registry instance.
func NewRegistry(db *pgxpool.Pool) *Registry {
	return &Registry{db: db}
}

func (r *Registry) CreateFarmer(ctx context.Context, f models.Farmer) error {
	query := `
		INSERT INTO farmers (id, name, phone, device_tier, locale, fcm_token)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err := r.db.Exec(ctx, query, f.ID, f.Name, f.Phone, f.DeviceTier, f.Locale, f.FCMToken)
	if err != nil {
		return fmt.Errorf("failed to insert farmer %s: %w", f.ID, err)
	}

	return nil
}

func (r *Registry) GetFarmer(ctx context.Context, id string) (*models.Farmer, error) {
	var f models.Farmer

	query := `
		SELECT id, name, phone, device_tier, locale, fcm_token 
		FROM farmers 
		WHERE id = $1
	`

	err := r.db.QueryRow(ctx, query, id).Scan(
		&f.ID,
		&f.Name,
		&f.Phone,
		&f.DeviceTier,
		&f.Locale,
		&f.FCMToken,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get farmer %s: %w", id, err)
	}

	return &f, nil
}

func (r *Registry) CreateFarm(ctx context.Context, f models.Farm) error {

	query := `
		INSERT INTO farms (id, farmer_id,name, dry_season_start, dry_season_end)
		VALUES ($1, $2, $3, $4, $5)
	`

	_, err := r.db.Exec(ctx, query, f.ID, f.FarmerID, f.Name, f.DrySeasonStart, f.DrySeasonEnd)
	if err != nil {
		return fmt.Errorf("failed to insert farm %s: %w", f.ID, err)
	}

	return nil
}

func (r *Registry) GetFarm(ctx context.Context, id string) (*models.Farm, error) {
	var f models.Farm

	query := `
		SELECT id, farmer_id, name, dry_season_start, dry_season_end 
		FROM farms 
		WHERE id = $1
	`

	err := r.db.QueryRow(ctx, query, id).Scan(
		&f.ID,
		&f.FarmerID,
		&f.Name,
		&f.DrySeasonStart,
		&f.DrySeasonEnd,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get farm %s: %w", id, err)
	}

	return &f, nil
}

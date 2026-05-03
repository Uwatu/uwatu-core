package farm

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/uwatu/uwatu-core/internal/models"
)

// AnimalRegistry handles database operations for Animals and Tags.
type AnimalRegistry struct {
	db *pgxpool.Pool
}

func NewAnimalRegistry(db *pgxpool.Pool) *AnimalRegistry {
	return &AnimalRegistry{db: db}
}

func (r *AnimalRegistry) CreateAnimal(ctx context.Context, a models.Animal) error {

	query := `
		INSERT INTO animals (id, farm_id, species)
		VALUES ($1, $2, $3)
	`

	_, err := r.db.Exec(ctx, query, a.ID, a.FarmID, a.Species)
	if err != nil {
		return fmt.Errorf("failed to insert animal %s: %w", a.ID, err)
	}
	return nil
}

func (r *AnimalRegistry) AssignTag(ctx context.Context, t models.Tag) error {
	query := `
			INSERT INTO tags (device_id, animal_id)
			VALUES ($1, $2)`

	_, err := r.db.Exec(ctx, query, t.DeviceID, t.AnimalID)
	if err != nil {
		return fmt.Errorf("failed to insert tag %s: %w", t.DeviceID, err)
	}
	return nil
}

func (r *AnimalRegistry) GetAnimalByDeviceID(ctx context.Context, deviceID string) (*models.Animal, error) {
	var a models.Animal

	query := `
		SELECT a.id, a.farm_id, a.species 
		FROM animals a 
		JOIN tags t ON a.id = t.animal_id 
		WHERE t.device_id = $1
	`

	err := r.db.QueryRow(ctx, query, deviceID).Scan(
		&a.ID,
		&a.FarmID,
		&a.Species,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get animal for device %s: %w", deviceID, err)
	}

	return &a, nil
}

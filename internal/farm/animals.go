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

// GetAnimal fetches a specific animal by its ID.
func (r *AnimalRegistry) GetAnimal(ctx context.Context, id string) (*models.Animal, error) {
	var a models.Animal

	query := `
		SELECT id, farm_id, species 
		FROM animals 
		WHERE id = $1
	`

	err := r.db.QueryRow(ctx, query, id).Scan(&a.ID, &a.FarmID, &a.Species)
	if err != nil {
		return nil, fmt.Errorf("failed to get animal %s: %w", id, err)
	}

	return &a, nil
}

// UpdateAnimal modifies an existing animal's farm assignment or species.
func (r *AnimalRegistry) UpdateAnimal(ctx context.Context, a models.Animal) error {
	query := `
		UPDATE animals 
		SET farm_id = $2, species = $3 
		WHERE id = $1
	`

	_, err := r.db.Exec(ctx, query, a.ID, a.FarmID, a.Species)
	if err != nil {
		return fmt.Errorf("failed to update animal %s: %w", a.ID, err)
	}

	return nil
}

// DeleteAnimal removes an animal and cascades the deletion to its assigned tags safely.
func (r *AnimalRegistry) DeleteAnimal(ctx context.Context, id string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	// If tx.Commit() is successful, this Rollback safely does nothing.
	defer tx.Rollback(ctx)

	deleteTagsQuery := `DELETE FROM tags WHERE animal_id = $1`
	_, err = tx.Exec(ctx, deleteTagsQuery, id)
	if err != nil {
		return fmt.Errorf("failed to delete associated tags for animal %s: %w", id, err)
	}

	deleteAnimalQuery := `DELETE FROM animals WHERE id = $1`
	_, err = tx.Exec(ctx, deleteAnimalQuery, id)
	if err != nil {
		return fmt.Errorf("failed to delete animal %s: %w", id, err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit deletion transaction: %w", err)
	}

	return nil
}

// RemoveTag unassigns a specific device tag from its animal.
func (r *AnimalRegistry) RemoveTag(ctx context.Context, deviceID string) error {
	query := `
		DELETE FROM tags 
		WHERE device_id = $1
	`

	_, err := r.db.Exec(ctx, query, deviceID)
	if err != nil {
		return fmt.Errorf("failed to remove tag %s: %w", deviceID, err)
	}

	return nil
}

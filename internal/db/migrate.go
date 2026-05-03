package db

import "github.com/uwatu/uwatu-core/internal/config"

func RunMigrations() {
	if Pool == nil {
		config.LogInfo("DB", "No pool, skipping migrations")
		return
	}
	// TODO: Run SQL files from /migrations directory
	config.LogInfo("DB", "Migrations skipped (not implemented)")
}

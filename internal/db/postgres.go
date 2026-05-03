package db

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/uwatu/uwatu-core/internal/config"
)

var Pool *pgxpool.Pool

func Connect() error {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		config.LogError("DB", "DATABASE_URL not set, skipping database")
		return nil // non-fatal — demo can run without DB
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return fmt.Errorf("unable to create pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return fmt.Errorf("unable to ping: %w", err)
	}

	Pool = pool
	config.LogInfo("DB", "Connected to PostgreSQL")
	return nil
}

func Close() {
	if Pool != nil {
		Pool.Close()
	}
}

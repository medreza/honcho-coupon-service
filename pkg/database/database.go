package database

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

func InitDB(ctx context.Context) (*pgxpool.Pool, error) {
	dbURL := os.Getenv("POSTGRESQL_URL")
	if dbURL == "" {
		dbURL = "postgres://admin:admin123@localhost:5432/coupon?sslmode=disable"
	}

	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		return nil, fmt.Errorf("unable to create connection pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("unable to ping database: %w", err)
	}

	if err := runMigrations(ctx, pool); err != nil {
		pool.Close()
		return nil, fmt.Errorf("unable to run migrations: %w", err)
	}

	return pool, nil
}

package database

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

var pool *pgxpool.Pool

func InitDB(ctx context.Context) error {
	dbURL := os.Getenv("POSTGRESQL_URL")
	if dbURL == "" {
		dbURL = "postgres://admin:admin123@localhost:5432/coupon?sslmode=disable"
	}

	var err error
	pool, err = pgxpool.New(ctx, dbURL)
	if err != nil {
		return fmt.Errorf("unable to create connection pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		return fmt.Errorf("unable to ping database: %w", err)
	}

	if err := runMigrations(ctx); err != nil {
		return fmt.Errorf("unable to run migrations: %w", err)
	}

	return nil
}

func GetPool() *pgxpool.Pool {
	return pool
}

func CloseDB() {
	if pool != nil {
		pool.Close()
	}
}

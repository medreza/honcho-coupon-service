package database

import (
	"context"
	"fmt"
)

func runMigrations(ctx context.Context) error {
	_, err := pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS coupons (
			id SERIAL PRIMARY KEY,
			name VARCHAR(255) UNIQUE NOT NULL,
			amount INT NOT NULL CHECK (amount >= 0),
			remaining_amount INT NOT NULL CHECK (remaining_amount >= 0),
			created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create coupons table: %w", err)
	}

	_, err = pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS claims (
			id SERIAL PRIMARY KEY,
			user_id VARCHAR(255) NOT NULL,
			coupon_name VARCHAR(255) NOT NULL REFERENCES coupons(name) ON DELETE CASCADE,
			claimed_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(user_id, coupon_name)
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create claims table: %w", err)
	}

	_, err = pool.Exec(ctx, `
		CREATE INDEX IF NOT EXISTS idx_claims_coupon_name ON claims(coupon_name)
	`)
	if err != nil {
		return fmt.Errorf("failed to create index: %w", err)
	}

	return nil
}

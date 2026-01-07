package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/medreza/honcho-coupon-service/pkg/models"
)

var (
	ErrCouponNotFound = errors.New("coupon not found")
	ErrCouponExists   = errors.New("coupon already exists")
	ErrAlreadyClaimed = errors.New("user already claimed this coupon")
	ErrNoStock        = errors.New("coupon is out of stock")
)

const pgUniqueViolationCode = "23505"

type CouponRepository struct {
	pool *pgxpool.Pool
}

func NewCouponRepository(pool *pgxpool.Pool) *CouponRepository {
	return &CouponRepository{pool: pool}
}

func (r *CouponRepository) CreateCoupon(ctx context.Context, name string, amount int) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO coupons (name, amount, remaining_amount) VALUES ($1, $2, $2)`,
		name, amount,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgUniqueViolationCode {
			return ErrCouponExists
		}
		return fmt.Errorf("failed to create coupon: %w", err)
	}
	return nil
}

func (r *CouponRepository) GetCouponByName(ctx context.Context, name string) (*models.Coupon, error) {
	var coupon models.Coupon
	err := r.pool.QueryRow(ctx,
		`SELECT id, name, amount, remaining_amount, created_at FROM coupons WHERE name = $1`,
		name,
	).Scan(&coupon.ID, &coupon.Name, &coupon.Amount, &coupon.RemainingAmount, &coupon.CreatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrCouponNotFound
		}
		return nil, fmt.Errorf("failed to get coupon: %w", err)
	}
	return &coupon, nil
}

func (r *CouponRepository) GetClaimsByCouponName(ctx context.Context, couponName string) ([]string, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT user_id FROM claims WHERE coupon_name = $1 ORDER BY claimed_at`,
		couponName,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get claims: %w", err)
	}
	defer rows.Close()

	var userIDs []string
	for rows.Next() {
		var userID string
		if err := rows.Scan(&userID); err != nil {
			return nil, fmt.Errorf("failed to scan claim: %w", err)
		}
		userIDs = append(userIDs, userID)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating claims: %w", err)
	}

	return userIDs, nil
}

func (r *CouponRepository) ClaimCoupon(ctx context.Context, userID, couponName string) error {
	// use transaction to make the cupon claiming process to be atomic
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// lock the row using 'FOR UPDATE' to prevent race condition when claiming the coupon
	var remainingAmount int
	err = tx.QueryRow(ctx,
		`SELECT remaining_amount FROM coupons WHERE name = $1 FOR UPDATE`,
		couponName,
	).Scan(&remainingAmount)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrCouponNotFound
		}
		return fmt.Errorf("failed to lock coupon: %w", err)
	}

	if remainingAmount <= 0 {
		return ErrNoStock
	}

	_, err = tx.Exec(ctx,
		`INSERT INTO claims (user_id, coupon_name) VALUES ($1, $2)`,
		userID, couponName,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) &&
			pgErr.Code == pgUniqueViolationCode {
			return ErrAlreadyClaimed
		}
		return fmt.Errorf("failed to insert claim: %w", err)
	}

	_, err = tx.Exec(ctx,
		`UPDATE coupons SET remaining_amount = remaining_amount - 1 WHERE name = $1`,
		couponName,
	)
	if err != nil {
		return fmt.Errorf("failed to update stock: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

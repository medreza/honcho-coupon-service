package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/medreza/honcho-coupon-service/pkg/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	ErrCouponNotFound = errors.New("coupon not found")
	ErrCouponExists   = errors.New("coupon already exists")
	ErrAlreadyClaimed = errors.New("user already claimed this coupon")
	ErrNoStock        = errors.New("coupon is out of stock")
)

const (
	couponsCollection = "coupons"
	claimsCollection  = "claims"
)

type CouponRepository struct {
	client *mongo.Client
	db     *mongo.Database
}

func NewCouponRepository(client *mongo.Client) *CouponRepository {
	db := client.Database("coupon")
	repo := &CouponRepository{
		client: client,
		db:     db,
	}

	repo.initIndexes(context.Background())

	return repo
}

func (r *CouponRepository) initIndexes(ctx context.Context) {
	// Unique index for coupon name
	_, _ = r.db.Collection(couponsCollection).Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "name", Value: 1}},
		Options: options.Index().SetUnique(true),
	})

	// Unique index for user_id and coupon_name to prevent multiple claims of same user-coupon pair
	_, _ = r.db.Collection(claimsCollection).Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{
			{Key: "user_id", Value: 1},
			{Key: "coupon_name", Value: 1},
		},
		Options: options.Index().SetUnique(true),
	})
}

func (r *CouponRepository) CreateCoupon(ctx context.Context, name string, amount int) error {
	coupon := models.Coupon{
		Name:            name,
		Amount:          amount,
		RemainingAmount: amount,
		CreatedAt:       time.Now(),
	}

	_, err := r.db.Collection(couponsCollection).InsertOne(ctx, coupon)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return ErrCouponExists
		}
		return fmt.Errorf("failed to create coupon: %w", err)
	}
	return nil
}

func (r *CouponRepository) GetCouponByName(ctx context.Context, name string) (*models.Coupon, error) {
	var coupon models.Coupon
	err := r.db.Collection(couponsCollection).FindOne(ctx, bson.M{"name": name}).Decode(&coupon)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrCouponNotFound
		}
		return nil, fmt.Errorf("failed to get coupon: %w", err)
	}
	return &coupon, nil
}

func (r *CouponRepository) GetClaimsByCouponName(ctx context.Context, couponName string) ([]string, error) {
	cursor, err := r.db.Collection(claimsCollection).Find(ctx, bson.M{"coupon_name": couponName}, options.Find().SetSort(bson.D{{Key: "claimed_at", Value: 1}}))
	if err != nil {
		return nil, fmt.Errorf("failed to find claims: %w", err)
	}
	defer cursor.Close(ctx)

	var claims []models.Claim
	if err := cursor.All(ctx, &claims); err != nil {
		return nil, fmt.Errorf("failed to decode claims: %w", err)
	}

	userIDs := make([]string, len(claims))
	for i, claim := range claims {
		userIDs[i] = claim.UserID
	}

	return userIDs, nil
}

func (r *CouponRepository) ClaimCoupon(ctx context.Context, userID, couponName string) error {
	session, err := r.client.StartSession()
	if err != nil {
		return fmt.Errorf("failed to start session: %w", err)
	}
	defer session.EndSession(ctx)

	_, err = session.WithTransaction(ctx, func(sessionCtx mongo.SessionContext) (interface{}, error) {
		// Step 1: Check if coupin claimed already
		var existingClaim models.Claim
		err := r.db.Collection(claimsCollection).FindOne(sessionCtx, bson.M{"user_id": userID, "coupon_name": couponName}).Decode(&existingClaim)
		if err == nil {
			return nil, ErrAlreadyClaimed
		}
		if !errors.Is(err, mongo.ErrNoDocuments) {
			return nil, fmt.Errorf("failed to check existing claim: %w", err)
		}

		// Step 2: Check stock and decrement remaining amount
		filter := bson.M{"name": couponName, "remaining_amount": bson.M{"$gt": 0}}
		update := bson.M{"$inc": bson.M{"remaining_amount": -1}}

		result, err := r.db.Collection(couponsCollection).UpdateOne(sessionCtx, filter, update)
		if err != nil {
			return nil, fmt.Errorf("failed to update coupon stock: %w", err)
		}

		if result.MatchedCount == 0 {
			// Check if coupon exists or actually out of stock
			var coupon models.Coupon
			err := r.db.Collection(couponsCollection).FindOne(sessionCtx, bson.M{"name": couponName}).Decode(&coupon)
			if err != nil {
				if errors.Is(err, mongo.ErrNoDocuments) {
					return nil, ErrCouponNotFound
				}
				return nil, fmt.Errorf("failed to check coupon status: %w", err)
			}
			return nil, ErrNoStock
		}

		// Step 3: Insert claim
		claim := models.Claim{
			UserID:     userID,
			CouponName: couponName,
			ClaimedAt:  time.Now(),
		}

		_, err = r.db.Collection(claimsCollection).InsertOne(sessionCtx, claim)
		if err != nil {
			if mongo.IsDuplicateKeyError(err) {
				return nil, ErrAlreadyClaimed
			}
			return nil, fmt.Errorf("failed to insert claim: %w", err)
		}

		return nil, nil
	})

	if err != nil {
		if errors.Is(err, ErrAlreadyClaimed) || errors.Is(err, ErrCouponNotFound) || errors.Is(err, ErrNoStock) {
			return err
		}
		return fmt.Errorf("transaction failed: %w", err)
	}

	return nil
}

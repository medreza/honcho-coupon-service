package models

import (
	"time"
)

type Coupon struct {
	Name            string    `json:"name" bson:"name"`
	Amount          int       `json:"amount" bson:"amount"`
	RemainingAmount int       `json:"remaining_amount" bson:"remaining_amount"`
	CreatedAt       time.Time `json:"created_at" bson:"created_at"`
}

type Claim struct {
	UserID     string    `json:"user_id" bson:"user_id"`
	CouponName string    `json:"coupon_name" bson:"coupon_name"`
	ClaimedAt  time.Time `json:"claimed_at" bson:"claimed_at"`
}

type CreateCouponRequest struct {
	Name   string `json:"name" binding:"required"`
	Amount int    `json:"amount" binding:"required,min=1"`
}

type ClaimCouponRequest struct {
	UserID     string `json:"user_id" binding:"required"`
	CouponName string `json:"coupon_name" binding:"required"`
}

type CouponDetailsResponse struct {
	Name            string   `json:"name"`
	Amount          int      `json:"amount"`
	RemainingAmount int      `json:"remaining_amount"`
	ClaimedBy       []string `json:"claimed_by"`
}

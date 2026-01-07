package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/medreza/honcho-coupon-service/pkg/models"
	"github.com/medreza/honcho-coupon-service/pkg/repository"
	"github.com/sirupsen/logrus"
)

type CouponHandler struct {
	couponRepo *repository.CouponRepository
}

func NewCouponHandler(couponRepo *repository.CouponRepository) *CouponHandler {
	return &CouponHandler{couponRepo: couponRepo}
}

func (h *CouponHandler) CreateCoupon(c *gin.Context) {
	var req models.CreateCouponRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logrus.WithField("error", err).Warn("CreateCoupon: Invalid request body")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body: " + err.Error()})
		return
	}

	err := h.couponRepo.CreateCoupon(c.Request.Context(), req.Name, req.Amount)
	if err != nil {
		if errors.Is(err, repository.ErrCouponExists) {
			logrus.WithField("coupon_name", req.Name).Warn("CreateCoupon: Coupon already exists")
			c.JSON(http.StatusConflict, gin.H{"error": "Coupon already exists"})
			return
		}
		logrus.WithError(err).Error("CreateCoupon: Failed to create coupon")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create coupon"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Coupon created successfully", "name": req.Name})
}

func (h *CouponHandler) ClaimCoupon(c *gin.Context) {
	var req models.ClaimCouponRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logrus.WithField("error", err).Warn("ClaimCoupon: Invalid request body")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body: " + err.Error()})
		return
	}

	err := h.couponRepo.ClaimCoupon(c.Request.Context(), req.UserID, req.CouponName)
	if err != nil {
		log := logrus.WithFields(logrus.Fields{
			"user_id":     req.UserID,
			"coupon_name": req.CouponName,
		})

		switch {
		case errors.Is(err, repository.ErrCouponNotFound):
			log.Warn("ClaimCoupon: Coupon not found")
			c.JSON(http.StatusNotFound, gin.H{"error": "Coupon not found"})
		case errors.Is(err, repository.ErrAlreadyClaimed):
			log.Warn("ClaimCoupon: User has already claimed this coupon")
			c.JSON(http.StatusConflict, gin.H{"error": "User has already claimed this coupon"})
		case errors.Is(err, repository.ErrNoStock):
			log.Warn("ClaimCoupon: Coupon is out of stock")
			c.JSON(http.StatusConflict, gin.H{"error": "Coupon is out of stock"})
		default:
			log.WithError(err).Error("ClaimCoupon: Failed to claim coupon")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to claim coupon"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":     "Coupon claimed successfully",
		"user_id":     req.UserID,
		"coupon_name": req.CouponName,
	})
}

func (h *CouponHandler) GetCoupon(c *gin.Context) {
	name := c.Param("name")
	if name == "" {
		logrus.Warn("GetCoupon: Coupon name is required")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Coupon name is required"})
		return
	}

	coupon, err := h.couponRepo.GetCouponByName(c.Request.Context(), name)
	if err != nil {
		if errors.Is(err, repository.ErrCouponNotFound) {
			logrus.WithField("coupon_name", name).Warn("GetCoupon: Coupon not found")
			c.JSON(http.StatusNotFound, gin.H{"error": "Coupon not found"})
			return
		}
		logrus.WithField("coupon_name", name).WithError(err).Error("GetCoupon: Failed to get coupon")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get coupon"})
		return
	}

	claimedBy, err := h.couponRepo.GetClaimsByCouponName(c.Request.Context(), name)
	if err != nil {
		logrus.WithField("coupon_name", name).WithError(err).Error("GetCoupon: Failed to get claims")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get claims"})
		return
	}

	if claimedBy == nil {
		claimedBy = make([]string, 0)
	}

	response := models.CouponDetailsResponse{
		Name:            coupon.Name,
		Amount:          coupon.Amount,
		RemainingAmount: coupon.RemainingAmount,
		ClaimedBy:       claimedBy,
	}

	c.JSON(http.StatusOK, response)
}

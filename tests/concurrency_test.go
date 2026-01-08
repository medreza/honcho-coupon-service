package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync"
	"testing"
	"time"
)

func getBaseURL() string {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	return fmt.Sprintf("http://localhost:%s/api", port)
}

func TestConcurrency(t *testing.T) {
	t.Run("FlashSaleAttack", func(t *testing.T) {
		couponName := fmt.Sprintf("FLASH_%d", time.Now().UnixNano())
		stock := 5
		requests := 50

		// Seed coupun
		createBody, _ := json.Marshal(map[string]interface{}{
			"name":   couponName,
			"amount": stock,
		})
		resp, err := http.Post(getBaseURL()+"/coupons", "application/json", bytes.NewBuffer(createBody))
		if err != nil || resp.StatusCode != http.StatusCreated {
			t.Fatalf("Failed to create coupon: %v", err)
		}

		var wg sync.WaitGroup
		wg.Add(requests)

		for i := 0; i < requests; i++ {
			go func(userID int) {
				defer wg.Done()
				claimBody, _ := json.Marshal(map[string]string{
					"user_id":     fmt.Sprintf("user_%d", userID),
					"coupon_name": couponName,
				})
				http.Post(getBaseURL()+"/coupons/claim", "application/json", bytes.NewBuffer(claimBody))
			}(i)
		}
		wg.Wait()

		// Verify
		resp, err = http.Get(getBaseURL() + "/coupons/" + couponName)
		if err != nil {
			t.Fatalf("Failed to get coupon details: %v", err)
		}
		defer resp.Body.Close()

		var details struct {
			Remaining int      `json:"remaining_amount"`
			ClaimedBy []string `json:"claimed_by"`
		}
		json.NewDecoder(resp.Body).Decode(&details)

		if details.Remaining != 0 {
			t.Errorf("Expected 0 remaining stock, got %d", details.Remaining)
		}
		if len(details.ClaimedBy) != stock {
			t.Errorf("Expected %d claims, got %d", stock, len(details.ClaimedBy))
		}
	})

	t.Run("DoubleDipAttack", func(t *testing.T) {
		couponName := fmt.Sprintf("DOUBLE_%d", time.Now().UnixNano())
		stock := 10
		requests := 15
		userID := "attacker_user"

		// Seed coupun
		createBody, _ := json.Marshal(map[string]interface{}{
			"name":   couponName,
			"amount": stock,
		})
		resp, err := http.Post(getBaseURL()+"/coupons", "application/json", bytes.NewBuffer(createBody))
		if err != nil || resp.StatusCode != http.StatusCreated {
			t.Fatalf("Failed to create coupon: %v", err)
		}

		// The same user try to claim the coupon multiple times
		var wg sync.WaitGroup
		wg.Add(requests)

		for i := 0; i < requests; i++ {
			go func() {
				defer wg.Done()
				claimBody, _ := json.Marshal(map[string]string{
					"user_id":     userID,
					"coupon_name": couponName,
				})
				http.Post(getBaseURL()+"/coupons/claim", "application/json", bytes.NewBuffer(claimBody))
			}()
		}
		wg.Wait()

		// Verify
		resp, err = http.Get(getBaseURL() + "/coupons/" + couponName)
		if err != nil {
			t.Fatalf("Failed to get coupon details: %v", err)
		}
		defer resp.Body.Close()

		var couponDetails struct {
			Remaining int      `json:"remaining_amount"`
			ClaimedBy []string `json:"claimed_by"`
		}
		json.NewDecoder(resp.Body).Decode(&couponDetails)

		if couponDetails.Remaining != 9 {
			t.Errorf("Expected 9 remaining stock, got %d", couponDetails.Remaining)
		}
		if len(couponDetails.ClaimedBy) != 1 {
			t.Errorf("Expected 1 user claim, got %d", len(couponDetails.ClaimedBy))
		}
	})
}

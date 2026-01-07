package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/medreza/honcho-coupon-service/pkg/database"
	"github.com/medreza/honcho-coupon-service/pkg/handlers"
	"github.com/medreza/honcho-coupon-service/pkg/repository"
)

func main() {
	ctx := context.Background()

	if err := database.InitDB(ctx); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer database.CloseDB()

	couponRepo := repository.NewCouponRepository(database.GetPool())
	couponHandler := handlers.NewCouponHandler(couponRepo)

	router := gin.Default()
	api := router.Group("/api")
	{
		api.POST("/coupons", couponHandler.CreateCoupon)
		api.POST("/coupons/claim", couponHandler.ClaimCoupon)
		api.GET("/coupons/:name", couponHandler.GetCoupon)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: router,
	}

	go func() {
		log.Printf("Starting service on port %s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start service: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down service...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Service forced to shutdown: %v", err)
	}

	log.Println("Service exited")
}

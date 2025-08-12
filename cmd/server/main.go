package main

import (
	"context"
	"go-digital-wallet/internal/config"
	"go-digital-wallet/pkg/database"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	cfg := config.LoadConfig()
	appLogger := config.NewLogger()

	db, err := database.NewPostgresConnection(&cfg.Database)
	if err != nil {
		appLogger.WithError(err).Fatal("Failed to connect to database")
	}

	if err := database.RunMigrations(&cfg.Database, appLogger); err != nil {
		appLogger.Fatalf("Failed to run migrations: %v", err)
	}

	// Connect to Redis
	redisClient := database.ConnectRedis(&cfg.Redis, appLogger)
	defer redisClient.Close()

	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	validator := config.NewValidator()

	config.Bootstrap(&config.BootstrapConfig{
		DB:        db,
		App:       router,
		Log:       appLogger,
		Validate:  validator,
		JWTConfig: &cfg.JWT,
	})

	server := &http.Server{
		Addr:           ":" + cfg.Server.Port,
		Handler:        router,
		ReadTimeout:    time.Duration(cfg.Server.ReadTimeout) * time.Second,
		WriteTimeout:   time.Duration(cfg.Server.WriteTimeout) * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	// Start server in a goroutine
	go func() {
		appLogger.WithField("port", cfg.Server.Port).Info("Starting HTTP server")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			appLogger.WithError(err).Fatal("Failed to start server")
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	appLogger.Info("Shutting down server...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		appLogger.WithError(err).Error("Server forced to shutdown")
	} else {
		appLogger.Info("Server exited gracefully")
	}
}

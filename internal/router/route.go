package router

import (
	"go-digital-wallet/internal/handler"
	"go-digital-wallet/internal/middleware"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type RouteConfig struct {
	App              *gin.Engine
	AuthHandler      handler.AuthHandler
	WalletHandler    handler.WalletHandler
	AuthMiddleware   *middleware.AuthMiddleware
	LoggerMiddleware gin.HandlerFunc
}

func (c *RouteConfig) SetupRoute() {
	c.App.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "healthy",
			"timestamp": time.Now().Format(time.RFC3339),
			"service":   "digital-wallet-api",
		})
	})

	c.App.Use(c.LoggerMiddleware)

	v1 := c.App.Group("/api/v1")
	{
		// Auth routes (public)
		auth := v1.Group("/auth")
		{
			auth.POST("/register", c.AuthHandler.Register)
			auth.POST("/login", c.AuthHandler.Login)
		}
		// Wallet routes
		protected := v1.Group("/wallets")
		{
			protected.Use(c.AuthMiddleware.JWTAuth())
			{
				protected.POST("/", c.WalletHandler.CreateWallet)
				protected.GET("/balance", c.WalletHandler.GetBalance)
				protected.POST("/withdraw", c.WalletHandler.Withdraw)
				protected.POST("/deposit", c.WalletHandler.Deposit)
				protected.GET("/transactions", c.WalletHandler.GetTransactionHistory)
			}
		}
	}
}

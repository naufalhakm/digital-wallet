package config

import (
	"go-digital-wallet/internal/handler"
	"go-digital-wallet/internal/middleware"
	"go-digital-wallet/internal/repository"
	"go-digital-wallet/internal/router"
	"go-digital-wallet/internal/usecase"
	"go-digital-wallet/pkg/token"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type BootstrapConfig struct {
	DB        *gorm.DB
	Redis     *redis.Client
	App       *gin.Engine
	Log       *logrus.Logger
	Validate  *validator.Validate
	JWTConfig *JWTConfig
}

func Bootstrap(config *BootstrapConfig) {
	jwtManager := token.NewTokenManager(config.JWTConfig.SecretKey, config.JWTConfig.ExpirationTime)
	// setup repositories
	walletRepository := repository.NewWalletRepository(config.DB, config.Log)
	userRepository := repository.NewUserRepository(config.DB, config.Log)

	// setup use cases
	walletUseCase := usecase.NewWalletUsecase(walletRepository, config.Log, config.Redis)
	authUsecase := usecase.NewAuthUsecase(userRepository, config.Log, jwtManager)

	// setup handlers
	walletHandler := handler.NewWalletHandler(walletUseCase, config.Log, config.Validate)
	authHandler := handler.NewAuthHandler(authUsecase, config.Log, config.Validate)

	// setup middleware
	authMiddleware := middleware.NewAuthMiddleware(config.JWTConfig.SecretKey, config.Log, jwtManager)

	routeConfig := router.RouteConfig{
		App:            config.App,
		WalletHandler:  walletHandler,
		AuthHandler:    authHandler,
		AuthMiddleware: authMiddleware,
	}
	routeConfig.SetupRoute()
}

package database

import (
	"context"
	"go-digital-wallet/internal/config"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
)

func ConnectRedis(cfg *config.RedisConfig, log *logrus.Logger) *redis.Client {
	rdb := redis.NewClient(&redis.Options{
		Addr:         cfg.Host + ":" + cfg.Port,
		Password:     cfg.Password,
		DB:           cfg.DB,
		PoolSize:     10,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := rdb.Ping(ctx).Result()
	if err != nil {
		log.Errorf("Failed to connect to Redis: %v", err)
		return nil
	}

	log.Info("Successfully connected to Redis")
	return rdb
}

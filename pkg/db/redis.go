package db

import (
	"context"

	"tx/internal/config"

	"github.com/redis/go-redis/v9"
)

// NewRedisClient 创建Redis客户端
func NewRedisClient(cfg *config.Config) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Address,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})

	// 验证连接
	if err := client.Ping(context.Background()).Err(); err != nil {
		return nil, err
	}

	return client, nil
}

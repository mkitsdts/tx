package db

import (
	"context"
	"fmt"

	"tx/internal/config"

	"github.com/jackc/pgx/v5/pgxpool"
)

// NewPostgresClient 创建PostgreSQL连接池
func NewPostgresClient(cfg *config.Config) (*pgxpool.Pool, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Postgres.Host,
		cfg.Postgres.Port,
		cfg.Postgres.User,
		cfg.Postgres.Password,
		cfg.Postgres.DBName,
		cfg.Postgres.SSLMode,
	)

	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, err
	}

	// 连接池可以在这里进行更多配置
	config.MaxConns = 10

	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		return nil, err
	}

	// 验证连接
	if err := pool.Ping(context.Background()); err != nil {
		return nil, err
	}

	// 初始化向量扩展
	if _, err := pool.Exec(context.Background(), "CREATE EXTENSION IF NOT EXISTS vector"); err != nil {
		return nil, err
	}

	return pool, nil
}

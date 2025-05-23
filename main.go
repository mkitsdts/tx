package main

import (
	"context"
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"syscall"

	"tx/internal/config"
	"tx/internal/grpc"
	"tx/internal/service"
	"tx/pkg/db"
	"tx/pkg/logger"
	"tx/pkg/tracer"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

func main() {
	app := fx.New(
		// 提供各种依赖
		fx.Provide(
			// 配置
			config.NewConfig,
			// 日志
			logger.NewLogger,
			// PostgreSQL
			db.NewPostgresClient,
			// Redis
			db.NewRedisClient,
			// User服务
			service.NewUserService,
			// System服务
			service.NewSystemService,
			// Jaeger
			startJaeger,
			// gRPC服务器
			grpc.NewGRPCServer,
		),
		// 调用初始化函数
		fx.Invoke(
			// 启动gRPC服务器
			startGRPCServer,
			// 启动pprof服务
			startPprofServer,
		),
	)

	startCtx, cancel := context.WithTimeout(context.Background(), fx.DefaultTimeout)
	defer cancel()

	if err := app.Start(startCtx); err != nil {
		os.Exit(1)
	}

	// 等待中断信号优雅关闭服务器
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	stopCtx, cancel := context.WithTimeout(context.Background(), fx.DefaultTimeout)
	defer cancel()

	if err := app.Stop(stopCtx); err != nil {
		os.Exit(1)
	}
}

func startJaeger(lc fx.Lifecycle, logger *zap.Logger, cfg *config.Config) (trace.TracerProvider, error) {
	tp, err := tracer.InitJaeger(cfg, logger) // 调用 pkg/tracer/jaeger.go 中的 InitJaeger
	if err != nil {
		logger.Error("Failed to initialize Jaeger tracer provider", zap.Error(err))
		return nil, err // 如果初始化失败，则返回错误
	}

	if tp == nil {
		// Jaeger 未启用或初始化未返回 provider
		logger.Info("Jaeger provider not initialized (likely disabled).")
		return nil, nil // 返回 nil, nil，fx 不会尝试管理生命周期或注入 nil
	}

	// 设置为全局 TracerProvider
	otel.SetTracerProvider(tp)

	// 在应用关闭时优雅关闭TracerProvider
	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			logger.Info("Shutting down tracer provider")
			if err := tp.Shutdown(ctx); err != nil {
				logger.Error("Error shutting down tracer provider", zap.Error(err))
				return err
			}
			return nil
		},
	})

	return tp, nil // 返回 TracerProvider 实例
}

func startGRPCServer(lc fx.Lifecycle, server *grpc.Server, logger *zap.Logger, cfg *config.Config) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			listener, err := net.Listen("tcp", cfg.GRPC.Address)
			if err != nil {
				return err
			}

			logger.Info("Starting gRPC server", zap.String("address", cfg.GRPC.Address))
			go func() {
				if err := server.Serve(listener); err != nil {
					logger.Error("Failed to start gRPC server", zap.Error(err))
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			logger.Info("Stopping gRPC server")
			server.GracefulStop()
			return nil
		},
	})
}

func startPprofServer(lc fx.Lifecycle, logger *zap.Logger, cfg *config.Config) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			logger.Info("Starting pprof server", zap.String("address", cfg.Pprof.Address))
			go func() {
				if err := http.ListenAndServe(cfg.Pprof.Address, nil); err != nil {
					logger.Error("Failed to start pprof server", zap.Error(err))
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			// 此处为关闭pprof服务的连接逻辑
			logger.Info("Pprof server will be closed with the application")
			return nil
		},
	})
}

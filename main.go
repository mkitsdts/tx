package main

import (
	"context"
	"net"
	"os"
	"os/signal"
	"syscall"

	"tx/internal/config"
	"tx/internal/grpc"
	"tx/internal/service"
	"tx/pkg/db"
	"tx/pkg/logger"
	"tx/pkg/tracer"

	tracesdk "go.opentelemetry.io/otel/sdk/trace"

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
			// Jaeger链路追踪
			tracer.InitJaeger,
			// PostgreSQL
			db.NewPostgresClient,
			// Redis
			db.NewRedisClient,
			// User服务
			service.NewUserService,
			// System服务
			service.NewSystemService,
			// gRPC服务器
			grpc.NewGRPCServer,
		),
		// 调用初始化函数
		fx.Invoke(
			// 启动gRPC服务器
			startGRPCServer,
			func(tp *tracesdk.TracerProvider, log *zap.Logger, cfg *config.Config) {
				// 这个日志会在 tracer.InitJaeger 成功执行后打印
				if tp != nil {
					log.Info("OTel TracerProvider successfully initialized and injected via fx.",
						zap.String("configured_jaeger_endpoint", cfg.Jaeger.Endpoint))
				} else {
					// 或者如果 InitJaeger 返回了 error，fx 可能不会调用这个 invoke，或者 tp 会是 nil
					log.Warn("OTel TracerProvider is nil after fx.Invoke. Check InitJaeger logic and configuration.",
						zap.String("configured_jaeger_endpoint", cfg.Jaeger.Endpoint))
				}
			},
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

func startGRPCServer(lc fx.Lifecycle, server *grpc.Server, logger *zap.Logger, cfg *config.Config) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			listener, err := net.Listen("tcp", cfg.GRPC.Address)
			if err != nil {
				return err
			}
			logger.Info("config jaeger", zap.String("endpoint", cfg.Jaeger.Endpoint))
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

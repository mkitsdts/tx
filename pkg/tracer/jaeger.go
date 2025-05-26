package tracer

import (
	"context"
	"tx/internal/config"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// InitJaeger 初始化Jaeger TracerProvider和Exporter
func InitJaeger(lc fx.Lifecycle, cfg *config.Config, logger *zap.Logger) (*tracesdk.TracerProvider, error) {
	// 创建Jaeger Exporter
	exporter, err := jaeger.New(jaeger.WithCollectorEndpoint(
		jaeger.WithEndpoint(cfg.Jaeger.Endpoint), // Jaeger收集器端点
	))
	if err != nil {
		logger.Error("Failed to create Jaeger exporter", zap.Error(err))
		return nil, err
	}

	// 创建TracerProvider
	tp := tracesdk.NewTracerProvider(
		tracesdk.WithBatcher(exporter), // 使用批量导出器提高性能
		tracesdk.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String(cfg.Jaeger.ServiceName), // 服务名 - 会在Jaeger UI中显示
			semconv.ServiceVersionKey.String("1.0.0"),             // 服务版本
			semconv.DeploymentEnvironmentKey.String("dev"),        // 环境
		)), // 附加服务信息
		tracesdk.WithSampler(tracesdk.AlwaysSample()), // 100%采样（生产环境可调整）
	)
	logger.Info("Jaeger exporter created successfully")

	// 设置为全局TracerProvider！
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{})) // 添加这一行

	// 注册生命周期钩子
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			logger.Info("Jaeger TracerProvider initialized successfully",
				zap.String("service_name", cfg.Jaeger.ServiceName),
				zap.String("jaeger_endpoint", cfg.Jaeger.Endpoint))
			return nil
		},
		OnStop: func(ctx context.Context) error {
			logger.Info("Shutting down Jaeger TracerProvider")
			return tp.Shutdown(ctx)
		},
	})

	return tp, nil
}

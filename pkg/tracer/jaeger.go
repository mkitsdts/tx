package tracer

import (
	"fmt"

	"tx/internal/config"

	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0" // 使用较新或特定版本的semconv
	"go.uber.org/zap"
)

// InitJaeger 根据配置初始化并返回 Jaeger TracerProvider
// 此版本适配现有的 config.JaegerConfig 结构
func InitJaeger(cfg *config.Config, logger *zap.Logger) (*sdktrace.TracerProvider, error) {
	// 检查 Jaeger 配置是否存在且 Endpoint 是否已配置
	if cfg.Jaeger.Endpoint == "" {
		logger.Info("Jaeger endpoint is not configured. Jaeger tracing will be disabled.")
		// 返回一个 no-op provider 或 nil, nil
		// return sdktrace.NewTracerProvider(), nil // No-op provider
		return nil, nil // 表示未初始化
	}

	// 使用 cfg.Jaeger.Endpoint 配置 Jaeger exporter
	logger.Info("Initializing Jaeger exporter with collector endpoint", zap.String("endpoint", cfg.Jaeger.Endpoint))
	exporter, err := jaeger.New(jaeger.WithCollectorEndpoint(jaeger.WithEndpoint(cfg.Jaeger.Endpoint)))
	if err != nil {
		return nil, fmt.Errorf("failed to create Jaeger exporter: %w", err)
	}

	// 使用 cfg.Jaeger.ServiceName 作为服务名
	// 如果你的顶层 Config 结构中没有 ServiceVersion 或 Environment，
	// 你可以在这里使用默认值或省略它们。
	serviceName := cfg.Jaeger.ServiceName
	if serviceName == "" {
		// 如果配置文件中也没有 service_name，则提供一个默认值
		serviceName = "unknown-service"
		logger.Warn("Jaeger service_name not configured, using default.", zap.String("default_service_name", serviceName))
	}

	// 创建资源 (Resource)
	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String(serviceName),
			// 如果需要，可以为 ServiceVersion 和 Environment 提供默认值或从其他地方获取
			// semconv.ServiceVersionKey.String("1.0.0"),          // 示例默认版本
			// semconv.DeploymentEnvironmentKey.String("development"), // 示例默认环境
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// 创建 TracerProvider
	// 为了开发和测试，可以使用 AlwaysSample。在生产环境中，考虑使用 TraceIDRatioBased Sampler。
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)

	logger.Info("Jaeger TracerProvider initialized successfully.", zap.String("service_name", serviceName))
	return tp, nil
}

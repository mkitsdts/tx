package grpc

import (
	pb "tx/proto/gen"

	"tx/internal/interceptor"
	"tx/internal/service"

	"go.uber.org/zap"
	"google.golang.org/grpc"
)

// Server 是gRPC服务器的包装
type Server struct {
	*grpc.Server
}

// NewGRPCServer 创建并配置gRPC服务器
func NewGRPCServer(userSvc *service.UserService, systemSvc *service.SystemService, logger *zap.Logger) *Server {
	// 创建拦截器
	authInterceptor := interceptor.NewAuthInterceptor(logger)
	tracerInterceptor := interceptor.NewTracerInterceptor(logger)

	// 创建gRPC服务器，注册所有拦截器
	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			tracerInterceptor.Unary(),
			authInterceptor.Unary(),
		),
		grpc.ChainStreamInterceptor(
			tracerInterceptor.Stream(),
			authInterceptor.Stream(),
		),
	)

	// 注册服务
	pb.RegisterUserServiceServer(grpcServer, userSvc)
	pb.RegisterSystemServiceServer(grpcServer, systemSvc)

	return &Server{Server: grpcServer}
}

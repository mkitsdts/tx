package interceptor

import (
	"context"

	"tx/pkg/utils"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// 上下文键类型
type contextKey string

// UserIDKey 是用户ID在上下文中的键
const UserIDKey contextKey = "user_id"

// AuthInterceptor 实现认证拦截器
type AuthInterceptor struct {
	logger *zap.Logger
}

// NewAuthInterceptor 创建认证拦截器
func NewAuthInterceptor(logger *zap.Logger) *AuthInterceptor {
	return &AuthInterceptor{
		logger: logger,
	}
}

// Unary 一元RPC的认证拦截器
func (i *AuthInterceptor) Unary() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		// 跳过不需要认证的方法，如登录和注册
		if isPublicMethod(info.FullMethod) {
			return handler(ctx, req)
		}

		// 执行认证
		userID, err := i.authenticate(ctx)
		if err != nil {
			return nil, err
		}

		// 将用户ID添加到上下文
		newCtx := context.WithValue(ctx, UserIDKey, userID)
		return handler(newCtx, req)
	}
}

// Stream 流式RPC的认证拦截器
func (i *AuthInterceptor) Stream() grpc.StreamServerInterceptor {
	return func(
		srv any,
		ss grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		if isPublicMethod(info.FullMethod) {
			return handler(srv, ss)
		}

		// 执行认证
		userID, err := i.authenticate(ss.Context())
		if err != nil {
			return err
		}

		// 包装流，将用户ID添加到上下文
		wrappedStream := &wrappedServerStream{
			ServerStream: ss,
			ctx:          context.WithValue(ss.Context(), UserIDKey, userID),
		}
		return handler(srv, wrappedStream)
	}
}

// authenticate 验证JWT并提取用户ID
func (i *AuthInterceptor) authenticate(ctx context.Context) (string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", status.Errorf(codes.Unauthenticated, "metadata is not provided")
	}

	authHeader := md["authorization"]
	if len(authHeader) == 0 {
		return "", status.Errorf(codes.Unauthenticated, "authorization token is not provided")
	}

	tokenStr := authHeader[0]
	claims, err := utils.ParseToken(tokenStr)
	if err != nil {
		return "", status.Errorf(codes.Unauthenticated, "invalid token: %v", err)
	}
	// 获取用户ID
	userID := claims.UserId
	if !ok {
		return "", status.Errorf(codes.Unauthenticated, "invalid token payload")
	}

	return userID, nil
}

// isPublicMethod 检查是否为公开方法（不需要认证）
func isPublicMethod(method string) bool {
	// 根据实际需求配置公开方法
	publicMethods := map[string]bool{
		"/user.UserService/Register": true,
		"/user.UserService/Login":    true,
	}
	return publicMethods[method]
}

// wrappedServerStream 包装grpc.ServerStream，使其使用自定义上下文
type wrappedServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

// Context 返回包含用户ID的上下文
func (w *wrappedServerStream) Context() context.Context {
	return w.ctx
}

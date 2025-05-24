package service

import (
	"context"
	"time"

	"tx/internal/config"
	"tx/pkg/utils"
	pb "tx/proto/gen"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// UserService 实现用户服务
type UserService struct {
	pb.UnimplementedUserServiceServer
	db     *pgxpool.Pool
	redis  *redis.Client
	logger *zap.Logger
	cfg    *config.Config
}

// NewUserService 创建用户服务
func NewUserService(db *pgxpool.Pool, redis *redis.Client, logger *zap.Logger, cfg *config.Config) *UserService {
	return &UserService{
		db:     db,
		redis:  redis,
		logger: logger,
		cfg:    cfg,
	}
}

// Register 用户注册
func (s *UserService) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	s.logger.Info("user register", zap.String("username", req.Username))
	// 检查参数是否合理
	if req.Username == "" || req.Password == "" {
		return nil, status.Error(codes.InvalidArgument, "username and password cannot be empty")
	}

	maxRetryTimes := 3
	// 检查用户是否已存在
	usernameKey := "register:" + req.Username
	// 添加重试机制
	for i := range maxRetryTimes {
		_, err := s.redis.Get(ctx, usernameKey).Result()
		if err == redis.Nil {
			break
		}
		// 指数退避
		durationTime := 1 << i
		if i > 0 {
			s.logger.Info("wait for register", zap.Int("wait_time", durationTime))
			time.Sleep(time.Duration(durationTime) * time.Millisecond)
		}
	}
	userId := utils.GenerateId()
	go s.EnsureRedisSet(ctx, usernameKey, userId, 0)
	go s.EnsureRedisSet(ctx, "login:"+req.Username, utils.EncryptPassword(req.Password), 0)
	for i := range maxRetryTimes {
		tx, err := s.db.Begin(ctx)
		if err != nil {
			s.logger.Error("begin transaction failed", zap.Error(err))
			continue
		}
		_, err = s.db.Exec(ctx, "INSERT INTO users (id, username, password) VALUES ($1, $2, $3)",
			userId, req.Username, req.Password)
		if err == nil {
			s.logger.Info("user register success", zap.String("username", req.Username))
			tx.Commit(ctx)
			break
		}
		// 指数退避
		durationTime := 1 << i
		if i > 0 {
			s.logger.Info("wait for register", zap.Int("wait_time", durationTime))
			time.Sleep(time.Duration(durationTime) * time.Millisecond)
		}
		if i == maxRetryTimes-1 {
			s.logger.Error("user register failed", zap.String("username", req.Username), zap.Error(err))
			tx.Rollback(ctx)
			return nil, status.Error(codes.Internal, "failed to register user")
		}
	}
	return &pb.RegisterResponse{
		Success: true,
		UserId:  userId,
	}, nil
}

// Login 用户登录
func (s *UserService) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	// 检查参数是否合理
	if req.Username == "" || req.Password == "" {
		return nil, status.Error(codes.InvalidArgument, "username and password cannot be empty")
	}
	// 检查用户是否存在
	key := "login:" + req.Username
	// 添加重试机制
	maxRetryTimes := 3
	var result string
	var err error
	for i := range maxRetryTimes {
		result, err = s.redis.Get(ctx, key).Result()
		if err == nil {
			break
		}
		// 指数退避
		durationTime := 1 << i
		if i > 0 {
			s.logger.Info("wait for login", zap.Int("wait_time", durationTime))
			time.Sleep(time.Duration(durationTime) * time.Millisecond)
		}
		if i == maxRetryTimes-1 {
			s.logger.Error("user login failed", zap.String("username", req.Username), zap.Error(err))
			return nil, status.Error(codes.Internal, "failed to login user")
		}
	}
	// 检查密码是否正确
	if result != utils.EncryptPassword(req.Password) {
		return nil, status.Error(codes.PermissionDenied, "password is incorrect")
	}
	jwt, err := utils.GenerateToken(req.Username)
	if err != nil {
		s.logger.Error("generate jwt failed", zap.String("username", req.Username), zap.Error(err))
		return &pb.LoginResponse{
			Success: false,
			Token:   "",
		}, status.Error(codes.Internal, "failed to generate jwt")
	}
	return &pb.LoginResponse{
		Success: true,
		Token:   jwt,
	}, nil
}

// GetUserInfo 获取用户信息
func (s *UserService) GetUserInfo(ctx context.Context, req *pb.GetUserInfoRequest) (*pb.GetUserInfoResponse, error) {
	// 检查参数是否合理
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "userId cannot be empty")
	}
	maxRetryTimes := 3
	var username string
	var likeEmbedding []float32
	for i := range maxRetryTimes {
		// 检查用户是否存在
		err := s.db.QueryRow(ctx, "SELECT username, like_embedding FROM users WHERE id = $1", req.UserId).Scan(&username, &likeEmbedding)
		if err == nil {
			s.logger.Info("user get info success", zap.String("userId", req.UserId))
			return &pb.GetUserInfoResponse{
				UserId:        req.UserId,
				Username:      username,
				LikeEmbedding: likeEmbedding,
			}, nil
		}
		// 指数退避
		durationTime := 1 << i
		if i > 0 {
			s.logger.Info("wait for get user info", zap.Int("wait_time", durationTime))
			time.Sleep(time.Duration(durationTime) * time.Millisecond)
		}
		if i == maxRetryTimes-1 {
			s.logger.Error("user get info failed", zap.String("userId", req.UserId), zap.Error(err))
			return nil, status.Error(codes.Internal, "failed to get user info")
		}
	}
	return nil, status.Error(codes.Internal, "failed to get user info")
}

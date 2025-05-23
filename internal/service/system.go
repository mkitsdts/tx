package service

import (
	"io"
	"os"

	"tx/internal/config"
	pb "tx/proto/gen"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// SystemService 实现系统服务
type SystemService struct {
	pb.UnimplementedSystemServiceServer
	logger *zap.Logger
	cfg    *config.Config
}

// NewSystemService 创建系统服务
func NewSystemService(logger *zap.Logger, cfg *config.Config) *SystemService {
	return &SystemService{
		logger: logger,
		cfg:    cfg,
	}
}

// SendFile 实现文件流式传输
func (s *SystemService) SendFile(req *pb.SendFileRequest, stream pb.SystemService_SendFileServer) error {
	// 打开文件
	file, err := os.Open(req.FilePath)
	if err != nil {
		s.logger.Error("Failed to open file", zap.String("path", req.FilePath), zap.Error(err))
		return status.Errorf(codes.Internal, "failed to open file: %v", err)
	}
	defer file.Close()

	// 缓冲区大小为1MB
	buffer := make([]byte, 1024*1024)

	for {
		// 读取文件内容
		n, err := file.Read(buffer)
		if err == io.EOF {
			break // 文件读取完成
		}
		if err != nil {
			s.logger.Error("Error reading file", zap.Error(err))
			return status.Errorf(codes.Internal, "error reading file: %v", err)
		}

		// 发送文件内容
		if err := stream.Send(&pb.FileChunk{
			Content: buffer[:n],
		}); err != nil {
			s.logger.Error("Error sending file chunk", zap.Error(err))
			return status.Errorf(codes.Internal, "error sending file chunk: %v", err)
		}
	}

	return nil
}

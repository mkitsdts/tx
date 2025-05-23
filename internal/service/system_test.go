package service

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"testing"

	"tx/internal/config"
	pb "tx/proto/gen" // Assuming this is the correct path to your generated protobuf code

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// mockSystem_SendFileServer is a mock implementation of pb.SystemService_SendFileServer
// It needs to satisfy the grpc.ServerStream interface as well.
type mockSystem_SendFileServer struct {
	SentChunks [][]byte
	errOnSend  error
	header     metadata.MD
	trailer    metadata.MD
	customCtx  context.Context
	// Explicitly not embedding grpc.ServerStream to ensure all methods are consciously implemented.
}

// Send stores the sent chunk.
func (m *mockSystem_SendFileServer) Send(chunk *pb.FileChunk) error {
	if m.errOnSend != nil {
		return m.errOnSend
	}
	// Copy the content as the buffer in the SUT might be reused.
	contentCopy := make([]byte, len(chunk.Content))
	copy(contentCopy, chunk.Content)
	m.SentChunks = append(m.SentChunks, contentCopy)
	return nil
}

// SetHeader is a mock implementation.
func (m *mockSystem_SendFileServer) SetHeader(md metadata.MD) error {
	m.header = metadata.Join(m.header, md)
	return nil
}

// SendHeader is a mock implementation.
func (m *mockSystem_SendFileServer) SendHeader(md metadata.MD) error {
	m.header = metadata.Join(m.header, md)
	// In a real server, this would send headers. For mock, just store.
	return nil
}

// SetTrailer is a mock implementation.
func (m *mockSystem_SendFileServer) SetTrailer(md metadata.MD) {
	m.trailer = metadata.Join(m.trailer, md)
}

// Context returns the context for this stream.
func (m *mockSystem_SendFileServer) Context() context.Context {
	if m.customCtx != nil {
		return m.customCtx
	}
	return context.Background()
}

// SendMsg is called by the generated code and is part of grpc.ServerStream.
func (m *mockSystem_SendFileServer) SendMsg(v interface{}) error {
	chunk, ok := v.(*pb.FileChunk)
	if !ok {
		return status.Errorf(codes.Internal, "mock SendMsg: unexpected type %T", v)
	}
	return m.Send(chunk)
}

// RecvMsg is part of grpc.ServerStream. Not used by SendFile from server side.
func (m *mockSystem_SendFileServer) RecvMsg(v interface{}) error {
	// For server-streaming, server does not receive messages this way.
	return io.EOF
}

func TestSystemService_SendFile(t *testing.T) {
	logger := zap.NewNop() // Use Nop logger for cleaner test output
	cfg := &config.Config{}

	// Helper to create a temporary file with given content
	createTempFile := func(t *testing.T, content []byte) string {
		t.Helper()
		// Use t.TempDir() to ensure automatic cleanup of the directory and its contents
		tmpFile, err := os.CreateTemp(t.TempDir(), "testfile-*.txt")
		require.NoError(t, err)
		if len(content) > 0 {
			_, err = tmpFile.Write(content)
			require.NoError(t, err)
		}
		err = tmpFile.Close()
		require.NoError(t, err)
		return tmpFile.Name()
	}

	t.Run("successful send - small file (single chunk)", func(t *testing.T) {
		service := NewSystemService(logger, cfg)
		fileContent := []byte("hello world from test")
		filePath := createTempFile(t, fileContent)

		mockStream := &mockSystem_SendFileServer{}
		req := &pb.SendFileRequest{FilePath: filePath}

		err := service.SendFile(req, mockStream)
		require.NoError(t, err)

		require.Len(t, mockStream.SentChunks, 1, "Expected one chunk for a small file")
		assert.Equal(t, fileContent, mockStream.SentChunks[0])
	})

	t.Run("successful send - large file (multiple chunks)", func(t *testing.T) {
		service := NewSystemService(logger, cfg)
		// SUT buffer is 1MB (1024*1024). Create content slightly larger.
		fileContent := make([]byte, 1024*1024+100)
		for i := range fileContent {
			fileContent[i] = byte(i % 256) // Fill with some pattern
		}
		filePath := createTempFile(t, fileContent)

		mockStream := &mockSystem_SendFileServer{}
		req := &pb.SendFileRequest{FilePath: filePath}

		err := service.SendFile(req, mockStream)
		require.NoError(t, err)

		// The SUT uses a 1MB buffer.
		// Chunk 1: 1MB. Chunk 2: 100 bytes. Total 2 chunks.
		require.Len(t, mockStream.SentChunks, 2, "Expected two chunks for a file of 1MB + 100 bytes")

		var receivedContent bytes.Buffer
		for _, chunkData := range mockStream.SentChunks {
			receivedContent.Write(chunkData)
		}
		assert.Equal(t, fileContent, receivedContent.Bytes(), "Reconstructed content should match original")
	})

	t.Run("successful send - empty file", func(t *testing.T) {
		service := NewSystemService(logger, cfg)
		filePath := createTempFile(t, []byte{}) // Empty file

		mockStream := &mockSystem_SendFileServer{}
		req := &pb.SendFileRequest{FilePath: filePath}

		err := service.SendFile(req, mockStream)
		require.NoError(t, err)
		assert.Empty(t, mockStream.SentChunks, "Expected no chunks for an empty file")
	})

	t.Run("file not found", func(t *testing.T) {
		service := NewSystemService(logger, cfg)
		mockStream := &mockSystem_SendFileServer{}
		req := &pb.SendFileRequest{FilePath: "nonexistent/file/path.txt"}

		err := service.SendFile(req, mockStream)
		require.Error(t, err)

		st, ok := status.FromError(err)
		require.True(t, ok, "Error should be a gRPC status error")
		assert.Equal(t, codes.Internal, st.Code())
		assert.Contains(t, st.Message(), "failed to open file")
	})

	t.Run("error on stream send", func(t *testing.T) {
		service := NewSystemService(logger, cfg)
		fileContent := []byte("data that will cause send error")
		filePath := createTempFile(t, fileContent)

		expectedErr := errors.New("simulated stream send error")
		mockStream := &mockSystem_SendFileServer{errOnSend: expectedErr}
		req := &pb.SendFileRequest{FilePath: filePath}

		err := service.SendFile(req, mockStream)
		require.Error(t, err)

		st, ok := status.FromError(err)
		require.True(t, ok, "Error should be a gRPC status error")
		assert.Equal(t, codes.Internal, st.Code())
		assert.Contains(t, st.Message(), "error sending file chunk")
		assert.Contains(t, st.Message(), expectedErr.Error()) // Check if original error is part of the message
	})
}

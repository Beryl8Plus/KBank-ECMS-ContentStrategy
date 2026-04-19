// Package middleware provides gRPC server interceptors that integrate with the
// project's custom logger (internal/infrastructure/logger). It exposes:
//   - Logging interceptors: log every call's method, duration, and gRPC status via LSystem / LError.
//   - Recovery interceptors: catch panics, log them via LError, and return codes.Internal to the caller.
package middleware

import (
	"context"
	"fmt"
	"runtime/debug"
	"strings"

	"github.com/google/uuid"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/recovery"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"kbank-ecms/internal/domain/entity"
	"kbank-ecms/internal/infrastructure/logger"
)

// grpcLogger adapts the project's custom logger to the logging.Logger interface
// required by go-grpc-middleware/v2.
type grpcLogger struct{}

// Log implements logging.Logger. It maps middleware log levels to project logger calls:
//   - ERROR → logger.LError
//   - INFO / WARNING / DEBUG → logger.LSystem with the matching Level string
func (g *grpcLogger) Log(ctx context.Context, level logging.Level, msg string, fields ...any) {
	correlationID := correlationIDFromContext(ctx)

	// Flatten key-value fields emitted by the interceptor into a readable suffix.
	var parts []string
	for i := 0; i+1 < len(fields); i += 2 {
		parts = append(parts, fmt.Sprintf("%v=%v", fields[i], fields[i+1]))
	}
	detail := msg
	if len(parts) > 0 {
		detail = msg + " | " + strings.Join(parts, " ")
	}

	if level == logging.LevelError {
		logger.LError(ctx, entity.ErrorLog{
			Service:       "CMS-RUNTIME-GRPC",
			CorrelationID: correlationID,
			Message:       detail,
		})
		return
	}

	lvl := "INFO"
	switch level {
	case logging.LevelWarn:
		lvl = "WARN"
	case logging.LevelDebug:
		lvl = "DEBUG"
	}

	logger.LSystem(ctx, entity.SystemLog{
		Service:       "CMS-RUNTIME-GRPC",
		Level:         lvl,
		CorrelationID: correlationID,
		Message:       detail,
	})
}

// correlationIDFromContext extracts the requestID from gRPC incoming metadata.
// Falls back to a generated UUID when the header is absent.
func correlationIDFromContext(ctx context.Context) string {
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if vals := md.Get("requestID"); len(vals) > 0 && vals[0] != "" {
			return vals[0]
		}
	}
	return uuid.New().String()
}

// LoggerUnaryInterceptor returns a unary server interceptor that logs every call.
func LoggerUnaryInterceptor() grpc.UnaryServerInterceptor {
	return logging.UnaryServerInterceptor(&grpcLogger{})
}

// LoggerStreamInterceptor returns a stream server interceptor that logs every call.
func LoggerStreamInterceptor() grpc.StreamServerInterceptor {
	return logging.StreamServerInterceptor(&grpcLogger{})
}

// recoveryHandler logs the panic via LError and returns a safe gRPC status to the caller.
func recoveryHandler(p any) error {
	logger.LError(context.Background(), entity.ErrorLog{
		Service:    "CMS-RUNTIME-GRPC",
		ErrorCode:  "PANIC",
		Message:    fmt.Sprintf("recovered from panic: %v", p),
		StackTrace: string(debug.Stack()),
	})
	return status.Errorf(codes.Internal, "internal server error")
}

// RecoveryUnaryInterceptor returns a unary server interceptor that recovers from panics.
func RecoveryUnaryInterceptor() grpc.UnaryServerInterceptor {
	return recovery.UnaryServerInterceptor(recovery.WithRecoveryHandler(recoveryHandler))
}

// RecoveryStreamInterceptor returns a stream server interceptor that recovers from panics.
func RecoveryStreamInterceptor() grpc.StreamServerInterceptor {
	return recovery.StreamServerInterceptor(recovery.WithRecoveryHandler(recoveryHandler))
}

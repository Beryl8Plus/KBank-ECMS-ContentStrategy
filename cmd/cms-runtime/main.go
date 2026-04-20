package main

import (
	"context"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"
	"google.golang.org/grpc"

	grpcserver "kbank-ecms/cmd/cms-runtime/internal/server"
	"kbank-ecms/internal/domain/entity"
	grpcmiddleware "kbank-ecms/internal/grpc/middleware"
	"kbank-ecms/internal/infrastructure/logger"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Load .env file if present (ignored in production where env vars are injected)
	if loadErr := godotenv.Load(); loadErr != nil {
		logger.LStartup(ctx, entity.StartupLog{
			Service: "MAIN",
			Level:   "WARN",
			Message: "No .env file found, relying on environment variables",
		})
	}

	logger.LStartup(ctx, entity.StartupLog{Service: "CMS-RUNTIME", Level: "INFO", Message: "Starting cms-runtime pod (pure gRPC evaluator)"})

	// Start gRPC server — stateless evaluation only (no DB, no Redis, no cache).
	grpcPort := os.Getenv("CMS_RUNTIME_GRPC_PORT")
	if grpcPort == "" {
		grpcPort = "50051"
	}
	lis, err := net.Listen("tcp", ":"+grpcPort)
	if err != nil {
		logger.LSystem(ctx, entity.SystemLog{Service: "CMS-RUNTIME", Level: "FATAL", Message: "gRPC listen failed: " + err.Error()})
		os.Exit(1)
	}
	grpcSrv := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			grpcmiddleware.RecoveryUnaryInterceptor(),
			grpcmiddleware.LoggerUnaryInterceptor(),
		),
		grpc.ChainStreamInterceptor(
			grpcmiddleware.RecoveryStreamInterceptor(),
			grpcmiddleware.LoggerStreamInterceptor(),
		),
	)
	grpcserver.Register(grpcSrv)
	go func() {
		if err := grpcSrv.Serve(lis); err != nil {
			logger.LSystem(context.Background(), entity.SystemLog{Service: "CMS-RUNTIME", Level: "FATAL", Message: "gRPC serve failed: " + err.Error()})
			os.Exit(1)
		}
	}()
	logger.LStartup(ctx, entity.StartupLog{
		Service: "CMS-RUNTIME",
		Level:   "INFO",
		Message: "gRPC server listening on :" + grpcPort,
	})

	// Wait for termination signal.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.LStartup(ctx, entity.StartupLog{Service: "CMS-RUNTIME", Level: "INFO", Message: "Shutting down..."})
	grpcSrv.GracefulStop()
	logger.LStartup(ctx, entity.StartupLog{Service: "CMS-RUNTIME", Level: "INFO", Message: "Stopped"})
}

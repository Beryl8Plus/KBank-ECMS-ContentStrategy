// Package testserver exposes the cms-runtime gRPC server registration for
// integration tests that live outside the cmd/cms-runtime tree.
package testserver

import (
	"google.golang.org/grpc"

	runtimeserver "kbank-ecms/cmd/cms-runtime/internal/server"
)

// Register attaches the RuntimeGRPCServer to a gRPC server instance.
// This is a thin wrapper intended only for integration-test use.
func Register(srv *grpc.Server) {
	runtimeserver.Register(srv)
}

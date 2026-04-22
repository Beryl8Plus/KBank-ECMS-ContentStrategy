// Package testserver exposes the svc-contstrat-runtime gRPC server registration for
// integration tests that live outside the cmd/svc-contstrat-runtime tree.
package testserver

import (
	"google.golang.org/grpc"

	runtimeserver "kbank-ecms/cmd/svc-contstrat-runtime/internal/server"
)

// Register attaches the RuntimeGRPCServer to a gRPC server instance.
// This is a thin wrapper intended only for integration-test use.
func Register(srv *grpc.Server) {
	runtimeserver.Register(srv)
}

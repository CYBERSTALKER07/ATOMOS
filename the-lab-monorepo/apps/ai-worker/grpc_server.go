// Package main — ai-worker service.
// The gRPC server section is appended here; see the top of this file for
// the existing Kafka consumer bootstrap (not shown — unchanged).

// ──────────────────────────────────────────────────────────────────────────
// gRPC Server — optimizer service
// Registers the lab.optimizer.v1.OptimizerService on :8082 so backend-go
// can reach it via "xds:///ai-worker" (mesh) or "host:8082" (direct).
// The existing HTTP server on :8081 remains for health checks.
// ──────────────────────────────────────────────────────────────────────────

package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"lab-ai-worker/optimizer"
	contract "optimizercontract"
)

const grpcPort = ":8082"

// solveServer implements optimizer.Handler.
// It is a thin wrapper that converts the gRPC call into the existing
// in-process optimizer.Solve() call — the same function the HTTP handler uses.
type solveServer struct{}

// Solve handles the gRPC RPC. Auth is the shared internal API key sent as
// gRPC metadata header "x-internal-api-key".
func (s *solveServer) Solve(ctx context.Context, req *contract.SolveRequest) (*contract.SolveResponse, error) {
	// Auth: pull x-internal-api-key from metadata.
	if internalAPIKey != "" {
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok || len(md.Get("x-internal-api-key")) == 0 || md.Get("x-internal-api-key")[0] != internalAPIKey {
			return nil, status.Error(codes.Unauthenticated, "missing or invalid x-internal-api-key")
		}
	}

	resp, err := runSolver(ctx, req)
	if err != nil {
		slog.ErrorContext(ctx, "grpc Solve failed", "err", err, "trace_id", req.TraceID)
		return nil, status.Errorf(codes.Internal, "solve: %v", err)
	}
	return resp, nil
}

// runSolver is the shared solve bridge for both the HTTP handler and the gRPC
// handler. It calls the in-process optimizer.Solve() so neither transport
// path re-implements business logic.
func runSolver(_ context.Context, req *contract.SolveRequest) (*contract.SolveResponse, error) {
	resp, err := optimizer.Solve(*req)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// startGRPCServer launches the gRPC server in a background goroutine.
// It is stopped when ctx is cancelled (SIGTERM path).
func startGRPCServer(ctx context.Context) error {
	lis, err := net.Listen("tcp", grpcPort)
	if err != nil {
		return fmt.Errorf("grpc listen %s: %w", grpcPort, err)
	}

	srv := grpc.NewServer(
		grpc.KeepaliveParams(keepalive.ServerParameters{
			MaxConnectionIdle:     60 * time.Second,
			MaxConnectionAge:      300 * time.Second,
			MaxConnectionAgeGrace: 10 * time.Second,
			Time:                  30 * time.Second,
			Timeout:               10 * time.Second,
		}),
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
			MinTime:             15 * time.Second,
			PermitWithoutStream: false,
		}),
		grpc.ChainUnaryInterceptor(
			unaryLoggingInterceptor,
			unaryAuthInterceptor,
		),
	)

	// Register the optimizer service using the shared descriptor.
	// optimizerServiceDesc is the grpc.ServiceDesc from rpc/optimizer/service.go,
	// replicated here to avoid an import cycle (ai-worker is a separate module).
	srv.RegisterService(&optimizerServiceDesc, &solveServer{})

	go func() {
		slog.Info("[GRPC] Optimizer service listening", "addr", grpcPort)
		if err := srv.Serve(lis); err != nil && ctx.Err() == nil {
			slog.Error("[GRPC] Server exited unexpectedly", "err", err)
		}
	}()

	go func() {
		<-ctx.Done()
		slog.Info("[GRPC] Graceful stop initiated")
		srv.GracefulStop()
	}()

	return nil
}

// ─── gRPC service descriptor (mirrors rpc/optimizer/service.go) ──────────
// Duplicated here to avoid a cross-module import; the ServiceName constant
// must stay in sync with the backend-go copy.

const optimizerServiceName = "lab.optimizer.v1.OptimizerService"

var optimizerServiceDesc = grpc.ServiceDesc{
	ServiceName: optimizerServiceName,
	HandlerType: (*interface {
		Solve(context.Context, *contract.SolveRequest) (*contract.SolveResponse, error)
	})(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Solve",
			Handler: func(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
				req := new(contract.SolveRequest)
				if err := dec(req); err != nil {
					return nil, err
				}
				type iface interface {
					Solve(context.Context, *contract.SolveRequest) (*contract.SolveResponse, error)
				}
				if interceptor == nil {
					return srv.(iface).Solve(ctx, req)
				}
				info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/" + optimizerServiceName + "/Solve"}
				return interceptor(ctx, req, info, func(ctx context.Context, req any) (any, error) {
					return srv.(iface).Solve(ctx, req.(*contract.SolveRequest))
				})
			},
		},
	},
	Streams: []grpc.StreamDesc{},
}

// ─── gRPC interceptors ───────────────────────────────────────────────────

func unaryLoggingInterceptor(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
	resp, err := handler(ctx, req)
	if err != nil {
		slog.ErrorContext(ctx, "grpc unary error", "method", info.FullMethod, "err", err)
	}
	return resp, err
}

func unaryAuthInterceptor(ctx context.Context, req any, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
	if internalAPIKey == "" {
		return handler(ctx, req)
	}
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok || len(md.Get("x-internal-api-key")) == 0 || md.Get("x-internal-api-key")[0] != internalAPIKey {
		return nil, status.Error(codes.Unauthenticated, "x-internal-api-key required")
	}
	return handler(ctx, req)
}

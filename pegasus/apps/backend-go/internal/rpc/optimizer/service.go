// Package optimizer is the internal gRPC service for the Phase 2 route
// optimiser. It wraps the existing contract.SolveRequest / SolveResponse
// types from optimizercontract with a gRPC service descriptor so both client
// and server can be registered against a *grpc.Server / *grpc.ClientConn.
//
// Wire format: JSON (via rpc.JSONCodec registered at init). The descriptor
// intentionally mirrors what protoc-gen-go-grpc would generate for:
//
//	service OptimizerService { rpc Solve(SolveRequest) returns (SolveResponse); }
//
// Caller usage:
//
//	// Server (ai-worker):
//	grpc.RegisterService(grpcSrv, &grpc.ServiceDesc{...}, &MyHandler{})
//
//	// Client (backend-go):
//	conn, _ := grpc.NewClient("xds:///ai-worker", grpc.WithCodec(rpc.JSONCodec{}))
//	client := NewClient(conn)
//	resp, err := client.Solve(ctx, req)
package optimizer

import (
	"context"

	"google.golang.org/grpc"

	contract "optimizercontract"
)

// ServiceName is the fully-qualified gRPC service name.
const ServiceName = "lab.optimizer.v1.OptimizerService"

// MethodSolve is the unary method path used by the client stub.
const MethodSolve = "/" + ServiceName + "/Solve"

// Handler must be implemented by the ai-worker gRPC server.
type Handler interface {
	Solve(ctx context.Context, req *contract.SolveRequest) (*contract.SolveResponse, error)
}

// Desc is the gRPC service descriptor — identical to what protoc would emit.
// Register it on a *grpc.Server via grpc.RegisterService.
var Desc = grpc.ServiceDesc{
	ServiceName: ServiceName,
	HandlerType: (*Handler)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Solve",
			Handler:    solveDelegateHandler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "optimizer/v1/service.grpc",
}

// solveDelegateHandler is the unary interceptor bridge between gRPC core and
// the Handler interface. It matches the signature grpc.MethodDesc.Handler
// expects.
func solveDelegateHandler(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
	req := new(contract.SolveRequest)
	if err := dec(req); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(Handler).Solve(ctx, req)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: MethodSolve}
	return interceptor(ctx, req, info, func(ctx context.Context, req any) (any, error) {
		return srv.(Handler).Solve(ctx, req.(*contract.SolveRequest))
	})
}

// ─── Client stub ──────────────────────────────────────────────────────────────

// Client is the backend-side stub. Construct via NewClient.
type Client struct {
	cc grpc.ClientConnInterface
}

// NewClient wraps an existing grpc.ClientConnInterface (usually a
// *grpc.ClientConn pointing at "xds:///ai-worker" for service-mesh routing,
// or "passthrough:///host:port" in dev/test).
func NewClient(cc grpc.ClientConnInterface) *Client {
	return &Client{cc: cc}
}

// Solve calls the remote optimiser. On any transport or service error it
// returns (nil, err) — the caller must fall back to the KMEANS_BINPACK path.
func (c *Client) Solve(ctx context.Context, req *contract.SolveRequest, opts ...grpc.CallOption) (*contract.SolveResponse, error) {
	resp := new(contract.SolveResponse)
	if err := c.cc.Invoke(ctx, MethodSolve, req, resp, opts...); err != nil {
		return nil, err
	}
	return resp, nil
}

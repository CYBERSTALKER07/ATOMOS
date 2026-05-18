// Package optimizergrpc is the gRPC replacement for dispatch/optimizerclient.
// It dials "xds:///ai-worker" when OPTIMIZER_GRPC_ADDR is set; otherwise it
// falls back to the HTTP client for backward compat.
//
// xDS mode: the blank import of google.golang.org/grpc/xds enables the xDS
// resolver. Google Cloud Service Mesh injects the xDS management server
// address via the GRPC_XDS_BOOTSTRAP env var. No application code change is
// needed to pick up traffic management rules (retries, circuit breaking,
// weight-based LB) configured in the mesh control plane.
//
// Fallback mode: OPTIMIZER_GRPC_ADDR="" → the caller continues to use the
// HTTP optimizerclient.Client, which already handles timeouts and degraded
// mode. This preserves the Phase 2 contract verbatim.
package optimizergrpc

import (
	"context"
	"fmt"
	"os"
	"time"

	"backend-go/cache"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/metadata"

	// Blank import enables xDS name resolver + load balancer.
	// When GRPC_XDS_BOOTSTRAP env var is not set, xDS resolves via passthrough.
	_ "google.golang.org/grpc/xds"

	"backend-go/internal/rpc"
	"backend-go/internal/rpc/optimizer"
	contract "optimizercontract"
)

const (
	// DefaultTimeout matches the HTTP client's 2.5 s budget.
	DefaultTimeout = 2500 * time.Millisecond
	dialTimeout    = 3 * time.Second

	// xDS service name — must match the mesh resource name in the
	// TrafficDirector / Cloud Service Mesh configuration.
	xdsTarget = "xds:///ai-worker"

	// Direct target used when xDS is not configured (dev / CI).
	envGRPCAddr      = "OPTIMIZER_GRPC_ADDR"
	envSidecarTarget = "OPTIMIZER_SIDE_ENDPOINT"

	// Trip quickly on transport degradation so callers can fall back without
	// burning the full timeout budget on repeated failing dials/calls.
	breakerFailureThreshold = 3
	breakerOpenDuration     = 5 * time.Second
)

var tracer = otel.Tracer("backend-go/internal/rpc/optimizergrpc")

type solverStub interface {
	Solve(ctx context.Context, req *contract.SolveRequest, opts ...grpc.CallOption) (*contract.SolveResponse, error)
}

// GRPCClient wraps the optimizer.Client with a managed *grpc.ClientConn.
// It is the replacement for dispatch/optimizerclient.Client when
// OPTIMIZER_GRPC_ADDR is set.
type GRPCClient struct {
	conn   *grpc.ClientConn
	stub   solverStub
	apiKey string
	cb     *cache.CircuitBreaker
}

// New dials the optimizer service. addr should be:
//   - "" → caller must use HTTP fallback (New returns nil, nil)
//   - "xds:///ai-worker" → service-mesh routing
//   - "host:port" → direct dial (dev/test)
func New(apiKey string) (*GRPCClient, error) {
	addr := os.Getenv(envSidecarTarget)
	if addr == "" {
		addr = os.Getenv(envGRPCAddr)
	}
	if addr == "" {
		return nil, nil // Signal: use HTTP fallback
	}
	if addr == "xds" {
		addr = xdsTarget
	}

	// Use insecure credentials within the cluster mesh (mTLS is handled by
	// the service mesh sidecar, not the application layer).
	dialCtx, cancelDial := context.WithTimeout(context.Background(), dialTimeout)
	defer cancelDial()

	conn, err := grpc.DialContext(dialCtx, addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
		grpc.WithDefaultCallOptions(
			grpc.ForceCodec(rpc.JSONCodec{}),
		),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                20 * time.Second,
			Timeout:             10 * time.Second,
			PermitWithoutStream: false,
		}),
		// gRPC retry policy: 3 attempts on UNAVAILABLE, exponential backoff.
		// Circuit breaking is delegated to the xDS control plane when in
		// mesh mode, so this is a lightweight safeguard for direct-dial mode.
		grpc.WithDefaultServiceConfig(`{
			"methodConfig": [{
				"name": [{"service": "pegasus.optimizer.v1.OptimizerService"}],
				"retryPolicy": {
					"maxAttempts": 3,
					"initialBackoff": "0.1s",
					"maxBackoff": "0.5s",
					"backoffMultiplier": 2,
					"retryableStatusCodes": ["UNAVAILABLE"]
				},
				"timeout": "2.5s"
			}]
		}`),
	)
	if err != nil {
		return nil, fmt.Errorf("optimizer grpc dial %s: %w", addr, err)
	}

	breaker := cache.NewCircuitBreaker("optimizer_grpc")
	breaker.FailureThreshold = breakerFailureThreshold
	breaker.OpenDuration = breakerOpenDuration

	return &GRPCClient{
		conn:   conn,
		stub:   optimizer.NewClient(conn),
		apiKey: apiKey,
		cb:     breaker,
	}, nil
}

// Solve calls the remote optimiser over gRPC with the shared timeout.
// On any error it returns (nil, err); callers can route through the existing
// optimizer fallback path (KMEANS_BINPACK) identically to HTTP degradation.
func (c *GRPCClient) Solve(ctx context.Context, req *contract.SolveRequest) (*contract.SolveResponse, error) {
	ctx, span := tracer.Start(ctx, "optimizer.Solve")
	defer span.End()

	ctx, cancel := context.WithTimeout(ctx, DefaultTimeout)
	defer cancel()

	// Inject the internal API key via gRPC metadata (mirrors X-Internal-Api-Key
	// header used by the HTTP client).
	ctx = metadata.AppendToOutgoingContext(ctx,
		"x-internal-api-key", c.apiKey,
		"x-trace-id", req.TraceID,
	)

	var resp *contract.SolveResponse
	call := func() error {
		result, solveErr := c.stub.Solve(ctx, req)
		if solveErr != nil {
			return solveErr
		}
		resp = result
		return nil
	}

	var err error
	if c.cb != nil {
		err = c.cb.Do(call)
	} else {
		err = call()
	}
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("optimizer grpc solve: %w", err)
	}

	span.SetStatus(codes.Ok, "")
	return resp, nil
}

// Close drains the connection. Call during graceful shutdown.
func (c *GRPCClient) Close() error {
	if c == nil || c.conn == nil {
		return nil
	}
	return c.conn.Close()
}

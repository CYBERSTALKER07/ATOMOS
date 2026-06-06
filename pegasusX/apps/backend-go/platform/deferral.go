package platform

import (
	"context"
	"fmt"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

// SessionChecker answers whether an actor has in-flight work that should defer forced updates.
type SessionChecker interface {
	HasActiveCriticalSession(ctx context.Context, actorID, actorRole string) (bool, string, error)
}

// SpannerSessionChecker inspects Orders, manifests, and payment sessions.
type SpannerSessionChecker struct {
	client *spanner.Client
}

// NewSpannerSessionChecker creates a Spanner-backed session checker.
func NewSpannerSessionChecker(client *spanner.Client) *SpannerSessionChecker {
	return &SpannerSessionChecker{client: client}
}

// HasActiveCriticalSession returns true when a hard update block would interrupt live ops.
func (c *SpannerSessionChecker) HasActiveCriticalSession(ctx context.Context, actorID, actorRole string) (bool, string, error) {
	if c.client == nil || actorID == "" {
		return false, "", nil
	}
	switch actorRole {
	case "DRIVER":
		return c.driverActive(ctx, actorID)
	case "RETAILER":
		return c.retailerActive(ctx, actorID)
	case "PAYLOAD", "FACTORY", "WAREHOUSE", "ADMIN":
		return c.manifestActive(ctx, actorID, actorRole)
	default:
		return false, "", nil
	}
}

func (c *SpannerSessionChecker) driverActive(ctx context.Context, driverID string) (bool, string, error) {
	stmt := spanner.Statement{
		SQL: `SELECT OrderId FROM Orders@{FORCE_INDEX=Idx_Orders_ByDriverCreated}
			WHERE DriverId = @did AND Status IN ('IN_TRANSIT','ARRIVED') LIMIT 1`,
		Params: map[string]any{"did": driverID},
	}
	return c.exists(ctx, stmt, "active_delivery")
}

func (c *SpannerSessionChecker) retailerActive(ctx context.Context, retailerID string) (bool, string, error) {
	stmt := spanner.Statement{
		SQL: `SELECT OrderId FROM Orders@{FORCE_INDEX=Idx_Orders_ByRetailerCreated}
			WHERE RetailerId = @rid AND Status IN ('IN_TRANSIT','ARRIVED','LOADED') LIMIT 1`,
		Params: map[string]any{"rid": retailerID},
	}
	ok, reason, err := c.exists(ctx, stmt, "active_order")
	if err != nil || ok {
		return ok, reason, err
	}
	payStmt := spanner.Statement{
		SQL: `SELECT SessionId FROM PaymentSessions
			WHERE RetailerId = @rid AND Status = 'AUTHORIZED' LIMIT 1`,
		Params: map[string]any{"rid": retailerID},
	}
	return c.exists(ctx, payStmt, "payment_pending")
}

func (c *SpannerSessionChecker) manifestActive(ctx context.Context, actorID, actorRole string) (bool, string, error) {
	var stmt spanner.Statement
	switch actorRole {
	case "FACTORY":
		stmt = spanner.Statement{
			SQL: `SELECT ManifestId FROM FactoryTruckManifests
				WHERE FactoryId = @fid AND State IN ('LOADING','SEALED') LIMIT 1`,
			Params: map[string]any{"fid": actorID},
		}
	default:
		stmt = spanner.Statement{
			SQL: `SELECT ManifestId FROM SupplierTruckManifests
				WHERE SupplierId = @sid AND State IN ('LOADING','SEALED') LIMIT 1`,
			Params: map[string]any{"sid": actorID},
		}
	}
	return c.exists(ctx, stmt, "manifest_active")
}

func (c *SpannerSessionChecker) exists(ctx context.Context, stmt spanner.Statement, reason string) (bool, string, error) {
	iter := c.client.Single().Query(ctx, stmt)
	defer iter.Stop()
	_, err := iter.Next()
	if err == iterator.Done {
		return false, "", nil
	}
	if err != nil {
		return false, "", fmt.Errorf("session check: %w", err)
	}
	return true, reason, nil
}

// NoopSessionChecker never defers updates (scaffold / tests).
type NoopSessionChecker struct{}

// HasActiveCriticalSession implements SessionChecker.
func (NoopSessionChecker) HasActiveCriticalSession(context.Context, string, string) (bool, string, error) {
	return false, "", nil
}

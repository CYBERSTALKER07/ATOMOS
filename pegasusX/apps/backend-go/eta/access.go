package eta

import (
	"context"
	"errors"
	"strings"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"google.golang.org/api/iterator"
)

// ErrRouteNotFound is returned for missing routes and unauthorized callers (no leak).
var ErrRouteNotFound = errors.New("route_not_found")

// RouteScope is durable ownership for a route id.
type RouteScope struct {
	RouteID     string
	SupplierID  string
	WarehouseID string
	DriverID    string
	RetailerIDs []string
}

// AllowRouteAccess is fail-closed ownership. Platform admin is not handled here.
func AllowRouteAccess(c auth.Claims, tenantSupplierID string, sc RouteScope) bool {
	if strings.TrimSpace(sc.RouteID) == "" {
		return false
	}
	switch c.Role {
	case auth.RoleAdmin, auth.RoleFactory, auth.RoleFactoryAdmin, auth.RolePayload:
		return supplierMatch(tenantSupplierID, c.SupplierID, sc.SupplierID)
	case auth.RoleWarehouse, auth.RoleWarehouseAdmin:
		if sc.WarehouseID != "" && auth.HomeNodeMatches(claimsCtx(c), sc.WarehouseID, auth.HomeNodeWarehouse) {
			return true
		}
		if c.HomeNodeType == auth.HomeNodeWarehouse && strings.TrimSpace(c.HomeNodeID) != "" {
			return strings.TrimSpace(c.HomeNodeID) == sc.WarehouseID
		}
		return supplierMatch(tenantSupplierID, c.SupplierID, sc.SupplierID)
	case auth.RoleDriver:
		did := strings.TrimSpace(c.Subject)
		if did != "" && did == sc.DriverID {
			return true
		}
		return strings.TrimSpace(c.HomeNodeID) != "" && c.HomeNodeID == sc.DriverID
	case auth.RoleRetailer:
		rid := strings.TrimSpace(auth.ResolveRetailerOrgID(c))
		if rid == "" {
			rid = strings.TrimSpace(c.Subject)
		}
		if rid == "" {
			return false
		}
		for _, id := range sc.RetailerIDs {
			if id == rid {
				return true
			}
		}
		return false
	default:
		return false
	}
}

func supplierMatch(tenantSID, claimSID, entitySID string) bool {
	want := strings.TrimSpace(entitySID)
	if want == "" {
		return false
	}
	if sid := strings.TrimSpace(tenantSID); sid != "" {
		return sid == want
	}
	return strings.TrimSpace(claimSID) == want
}

func claimsCtx(c auth.Claims) context.Context {
	return auth.WithClaims(context.Background(), c)
}

// AuthorizeRoute returns nil when the caller may see or write ETAs for routeID.
func (s *Service) AuthorizeRoute(ctx context.Context, routeID string) error {
	routeID = strings.TrimSpace(routeID)
	if routeID == "" {
		return ErrRouteNotFound
	}
	if auth.IsPlatformAdmin(ctx) {
		return nil
	}
	if _, ok := auth.FromContext(ctx); !ok {
		return ErrRouteNotFound
	}
	if s == nil || s.spanner == nil {
		return ErrRouteNotFound
	}
	sc, err := s.LoadRouteScope(ctx, routeID)
	if err != nil {
		return err
	}
	c, _ := auth.FromContext(ctx)
	tenant, _ := auth.CallerSupplierID(ctx)
	if !AllowRouteAccess(c, tenant, sc) {
		return ErrRouteNotFound
	}
	return nil
}

// LoadRouteScope merges RouteTwins+Drivers, truck manifests, and orders.
func (s *Service) LoadRouteScope(ctx context.Context, routeID string) (RouteScope, error) {
	sc := RouteScope{RouteID: strings.TrimSpace(routeID)}
	if s == nil || s.spanner == nil || sc.RouteID == "" {
		return RouteScope{}, nil
	}
	if err := s.fillScopeFromTwin(ctx, &sc); err != nil {
		return RouteScope{}, err
	}
	if err := s.fillScopeFromManifest(ctx, &sc); err != nil {
		return RouteScope{}, err
	}
	if err := s.fillScopeFromOrders(ctx, &sc); err != nil {
		return RouteScope{}, err
	}
	if sc.SupplierID == "" && sc.DriverID == "" && sc.WarehouseID == "" && len(sc.RetailerIDs) == 0 {
		return RouteScope{}, nil
	}
	return sc, nil
}

func (s *Service) fillScopeFromTwin(ctx context.Context, sc *RouteScope) error {
	iter := s.spanner.Single().Query(ctx, spanner.Statement{
		SQL: `SELECT rt.DriverId, d.SupplierId
		      FROM RouteTwins rt
		      JOIN Drivers d ON d.DriverId = rt.DriverId
		      WHERE rt.RouteId = @rid
		      LIMIT 1`,
		Params: map[string]any{"rid": sc.RouteID},
	})
	defer iter.Stop()
	row, err := iter.Next()
	if err == iterator.Done {
		return nil
	}
	if err != nil {
		return err
	}
	var driverID, supplierID string
	if err := row.Columns(&driverID, &supplierID); err != nil {
		return err
	}
	if sc.DriverID == "" {
		sc.DriverID = strings.TrimSpace(driverID)
	}
	if sc.SupplierID == "" {
		sc.SupplierID = strings.TrimSpace(supplierID)
	}
	return nil
}

func (s *Service) fillScopeFromManifest(ctx context.Context, sc *RouteScope) error {
	iter := s.spanner.Single().Query(ctx, spanner.Statement{
		SQL: `SELECT SupplierId, COALESCE(WarehouseId, ''), DriverId
		      FROM SupplierTruckManifests
		      WHERE RouteId = @rid
		      LIMIT 1`,
		Params: map[string]any{"rid": sc.RouteID},
	})
	defer iter.Stop()
	row, err := iter.Next()
	if err == iterator.Done {
		return nil
	}
	if err != nil {
		return err
	}
	var sid, wid, did string
	if err := row.Columns(&sid, &wid, &did); err != nil {
		return err
	}
	if sc.SupplierID == "" {
		sc.SupplierID = strings.TrimSpace(sid)
	}
	if sc.WarehouseID == "" {
		sc.WarehouseID = strings.TrimSpace(wid)
	}
	if sc.DriverID == "" {
		sc.DriverID = strings.TrimSpace(did)
	}
	return nil
}

func (s *Service) fillScopeFromOrders(ctx context.Context, sc *RouteScope) error {
	iter := s.spanner.Single().Query(ctx, spanner.Statement{
		SQL: `SELECT SupplierId, COALESCE(WarehouseId, ''), COALESCE(RetailerId, '')
		      FROM Orders
		      WHERE RouteId = @rid`,
		Params: map[string]any{"rid": sc.RouteID},
	})
	defer iter.Stop()
	seen := map[string]struct{}{}
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			return nil
		}
		if err != nil {
			return err
		}
		var sid, wid, rid string
		if err := row.Columns(&sid, &wid, &rid); err != nil {
			return err
		}
		if sc.SupplierID == "" {
			sc.SupplierID = strings.TrimSpace(sid)
		}
		if sc.WarehouseID == "" {
			sc.WarehouseID = strings.TrimSpace(wid)
		}
		if r := strings.TrimSpace(rid); r != "" {
			if _, ok := seen[r]; !ok {
				seen[r] = struct{}{}
				sc.RetailerIDs = append(sc.RetailerIDs, r)
			}
		}
	}
}

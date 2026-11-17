package e2ep0

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/audit"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/bootstrap"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/foundry"
	"github.com/pegasusx/pegasusx/apps/backend-go/foundryroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/ontology"
	"github.com/pegasusx/pegasusx/apps/backend-go/ontologyroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/planning"
	"github.com/pegasusx/pegasusx/apps/backend-go/supplier"
	"github.com/stretchr/testify/require"
	"google.golang.org/api/iterator"
)

func TestE2E_OntologyHTTP_RBAC_AuditMiddleware_And_UnifiedKG(t *testing.T) {
	client := openInventorySpannerClient(t)
	ctx := context.Background()

	suffix := uuid.NewString()[:8]
	cellID := "cell-e2e-ont-" + suffix
	supID := "sup-e2e-ont-" + suffix
	whID := "wh-e2e-ont-" + suffix
	drvID := "drv-e2e-ont-" + suffix
	retID := "ret-e2e-ont-" + suffix
	ordID := "ord-e2e-ont-" + suffix
	today := time.Now().UTC().Format("2006-01-02")

	// Seed live Spanner domain rows for RebuildFromSpanner & Unified KG verification.
	_, err := client.Apply(ctx, []*spanner.Mutation{
		spanner.InsertMap("Suppliers", map[string]any{
			"SupplierId": supID, "Name": "E2E Dairy", "CountryCode": "UZ", "Currency": "UZS",
			"IsConfigured": true, "CreatedAt": spanner.CommitTimestamp, "UpdatedAt": spanner.CommitTimestamp,
		}),
		spanner.InsertMap("Warehouses", map[string]any{
			"WarehouseId": whID, "SupplierId": supID, "Name": "E2E Cold Depot",
			"CountryCode": "UZ", "CoverageRadiusKm": float64(20.0), "IsActive": true, "IsOnShift": true, "AutoDispatchEnabled": true,
			"CreatedAt": spanner.CommitTimestamp, "UpdatedAt": spanner.CommitTimestamp,
		}),
		spanner.InsertMap("Drivers", map[string]any{
			"DriverId": drvID, "Name": "Confidential E2E Driver", "Phone": "+998901234567",
			"SupplierId": supID, "HomeNodeType": "WAREHOUSE", "HomeNodeId": whID,
			"IsActive": true, "OnShift": true, "CreatedAt": spanner.CommitTimestamp, "UpdatedAt": spanner.CommitTimestamp,
		}),
		spanner.InsertMap("Retailers", map[string]any{
			"RetailerId": retID, "Phone": "+998907778899", "Name": "E2E Baraka",
			"CountryCode": "UZ", "CreatedAt": spanner.CommitTimestamp,
		}),
		spanner.InsertMap("Orders", map[string]any{
			"OrderId": ordID, "SupplierId": supID, "RetailerId": retID, "WarehouseId": whID,
			"DriverId": drvID, "Status": "CONFIRMED", "OrderSource": "MOBILE", "ConfirmationStatus": "AUTO",
			"LineItemsJson": []byte("[]"), "TotalMinor": int64(750000), "Currency": "UZS", "Version": int64(1),
			"CreatedAt": spanner.CommitTimestamp, "UpdatedAt": spanner.CommitTimestamp,
		}),
	})
	require.NoError(t, err)

	t.Cleanup(func() {
		cleanCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		ro := client.ReadOnlyTransaction().WithTimestampBound(spanner.StrongRead())
		obIter := ro.Query(cleanCtx, spanner.Statement{
			SQL:    `SELECT EventId FROM OutboxEvents WHERE AggregateType = @agg AND STARTS_WITH(AggregateId, @prefix)`,
			Params: map[string]any{"agg": events.AggregateOntologyObject, "prefix": cellID + "/"},
		})
		var muts []*spanner.Mutation
		for {
			row, rerr := obIter.Next()
			if rerr == iterator.Done {
				break
			}
			if rerr == nil {
				var id string
				if row.Column(0, &id) == nil {
					muts = append(muts, spanner.Delete("OutboxEvents", spanner.Key{id}))
				}
			}
		}
		obIter.Stop()
		ro.Close()

		muts = append(muts,
			spanner.Delete("AuditEvents", spanner.KeyRange{Start: spanner.Key{supID, today}, End: spanner.Key{supID, today}, Kind: spanner.ClosedClosed}),
			spanner.Delete("OntologyGraphEdges", spanner.KeyRange{Start: spanner.Key{cellID}, End: spanner.Key{cellID}, Kind: spanner.ClosedClosed}),
			spanner.Delete("OntologyObjects", spanner.KeyRange{Start: spanner.Key{cellID}, End: spanner.Key{cellID}, Kind: spanner.ClosedClosed}),
			spanner.Delete("Orders", spanner.Key{ordID}),
			spanner.Delete("Retailers", spanner.Key{retID}),
			spanner.Delete("Drivers", spanner.Key{drvID}),
			spanner.Delete("Warehouses", spanner.Key{whID}),
			spanner.Delete("Suppliers", spanner.Key{supID}),
		)
		_, _ = client.Apply(cleanCtx, muts)
	})

	// 1. Verify TestHarness mounts /v1/ontology/* with AuditMiddleware on mutating routes.
	h := NewTestHarnessWithOptions(t, HarnessOptions{SpannerClient: client})
	refAuditMWPtr := reflect.ValueOf(audit.AuditMiddleware(h.AuditSvc)).Pointer()
	var sawRebuildRoute bool
	require.NoError(t, chi.Walk(h.Router, func(method, route string, _ http.Handler, middlewares ...func(http.Handler) http.Handler) error {
		if method == http.MethodPost && route == "/v1/ontology/rebuild" {
			sawRebuildRoute = true
			hasAudit := false
			for _, mw := range append(middlewares, h.Router.Middlewares()...) {
				if isAuditMiddlewareHandler(mw, refAuditMWPtr) {
					hasAudit = true
					break
				}
			}
			require.True(t, hasAudit, "/v1/ontology/rebuild must be covered by audit.AuditMiddleware")
		}
		return nil
	}))
	require.True(t, sawRebuildRoute, "expected /v1/ontology/rebuild mounted on TestHarness")

	// 2. Build integrated HTTP server combining ontologyroutes + foundryroutes + /v1/supplier/knowledge-graph.
	auditSvc := audit.NewService(client)
	ontSvc := ontology.NewService(ontology.NewSpannerRepository(client), nil)
	ontSvc.SetDefaultCell(cellID)
	supSvc := supplier.NewService(supplier.ServiceConfig{SupplierID: supID})
	supSvc.SetUnifiedGraph(planning.AdaptOntologyService(ontSvc, cellID))

	r := chi.NewRouter()
	r.Use(bootstrap.DevCORSMiddleware())
	r.Use(auth.SessionAuth(DefaultJWTSecret))
	r.Use(auth.AttachTenantFromClaims)
	r.Use(audit.AuditMiddleware(auditSvc))

	ontologyroutes.RegisterRoutes(r, ontologyroutes.Deps{Service: ontSvc, JWTSecret: DefaultJWTSecret})
	foundryroutes.RegisterRoutes(r, foundryroutes.Deps{Engine: foundry.NewEngine().WithSpannerClient(client)})
	r.Get("/v1/supplier/knowledge-graph", supSvc.HandleKnowledgeGraph)

	ts := httptest.NewServer(r)
	defer ts.Close()

	issueRoleTok := func(sub string, role auth.Role, supRole auth.Role, actorType string) string {
		tok, err := auth.Issue(auth.Claims{
			Subject:      sub,
			Role:         role,
			SupplierID:   supID,
			SupplierRole: supRole,
			ActorType:    actorType,
		}, auth.IssueOptions{Secret: DefaultJWTSecret, TTL: time.Hour, Issuer: DefaultJWTIssuer})
		require.NoError(t, err)
		return tok
	}

	ownerTok := issueRoleTok("owner-"+suffix, auth.RoleAdmin, auth.SupplierRoleOwner, auth.ActorTypeHuman)
	agentTok := issueRoleTok("agent-"+suffix, auth.RoleAdmin, auth.SupplierRoleAgent, auth.ActorTypeAgent)
	platTok := issueRoleTok("plat-"+suffix, auth.RolePlatformAdmin, "", auth.ActorTypeHuman)

	call := func(method, path string, body any, tok string) (int, []byte) {
		var buf bytes.Buffer
		if body != nil {
			require.NoError(t, json.NewEncoder(&buf).Encode(body))
		}
		req, reqErr := http.NewRequest(method, ts.URL+path, &buf)
		require.NoError(t, reqErr)
		req.Header.Set("Content-Type", "application/json")
		if tok != "" {
			req.Header.Set("Authorization", "Bearer "+tok)
		}
		resp, err := ts.Client().Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		var out bytes.Buffer
		_, _ = out.ReadFrom(resp.Body)
		return resp.StatusCode, out.Bytes()
	}

	// 3. Foundry direct ontology write blocked for human OWNER (403) and allowed for PLATFORM_ADMIN (201)
	st, _ := call(http.MethodPost, "/v1/foundry/ontology/objects", map[string]any{
		"cell_id": cellID, "supplier_id": supID, "object_type": "Warehouse", "object_id": "wh-blocked-" + suffix,
		"properties": map[string]any{"name": "Blocked"},
	}, ownerTok)
	require.Equal(t, http.StatusForbidden, st)

	st, _ = call(http.MethodPost, "/v1/foundry/ontology/objects", map[string]any{
		"cell_id": cellID, "supplier_id": supID, "object_type": "Warehouse", "object_id": "wh-plat-" + suffix,
		"properties": map[string]any{"name": "Platform Allowed"},
	}, platTok)
	require.Equal(t, http.StatusCreated, st)

	// 4. RebuildFromSpanner via HTTP POST /v1/ontology/rebuild: AGENT -> 403, OWNER -> 200
	st, _ = call(http.MethodPost, "/v1/ontology/rebuild", map[string]any{"cell_id": cellID, "supplier_id": supID}, agentTok)
	require.Equal(t, http.StatusForbidden, st)

	st, b := call(http.MethodPost, "/v1/ontology/rebuild", map[string]any{"cell_id": cellID, "supplier_id": supID}, ownerTok)
	require.Equal(t, http.StatusOK, st, "body=%s", string(b))
	var rb ontology.RebuildResult
	require.NoError(t, json.Unmarshal(b, &rb))
	require.Equal(t, "COMPLETED", rb.Status)
	require.GreaterOrEqual(t, rb.ObjectsUpserted, 5)

	// 5. GET /v1/ontology/objects?type=Order enforces security markings by viewer role
	st, b = call(http.MethodGet, fmt.Sprintf("/v1/ontology/objects?cell_id=%s&type=Order", cellID), nil, agentTok)
	require.Equal(t, http.StatusOK, st, "body=%s", string(b))
	var agentOrders ontology.ListObjectsResult
	require.NoError(t, json.Unmarshal(b, &agentOrders))
	require.True(t, agentOrders.FinanceMasked)
	require.Len(t, agentOrders.Objects, 1)
	require.Equal(t, ontology.RedactedPlaceholder, agentOrders.Objects[0].Properties["total_minor"])

	st, b = call(http.MethodGet, fmt.Sprintf("/v1/ontology/objects?cell_id=%s&type=Order", cellID), nil, ownerTok)
	require.Equal(t, http.StatusOK, st)
	var ownerOrders ontology.ListObjectsResult
	require.NoError(t, json.Unmarshal(b, &ownerOrders))
	require.False(t, ownerOrders.FinanceMasked)
	require.EqualValues(t, 750000, ownerOrders.Objects[0].Properties["total_minor"])

	// 6. GET /v1/ontology/objects/Order/{id}/neighbors?depth=2
	st, b = call(http.MethodGet, fmt.Sprintf("/v1/ontology/objects/Order/%s/neighbors?cell_id=%s&depth=2", ordID, cellID), nil, agentTok)
	require.Equal(t, http.StatusOK, st, "body=%s", string(b))
	var neigh ontology.NeighborResult
	require.NoError(t, json.Unmarshal(b, &neigh))
	require.True(t, neigh.PIIMasked)
	require.GreaterOrEqual(t, neigh.TotalNodes, 3)

	// 7. GET /v1/supplier/knowledge-graph serves unified ontology graph + redaction
	st, b = call(http.MethodGet, "/v1/supplier/knowledge-graph", nil, agentTok)
	require.Equal(t, http.StatusOK, st)
	var kgAgent planning.KnowledgeGraph
	require.NoError(t, json.Unmarshal(b, &kgAgent))
	require.Equal(t, "unified_ontology", kgAgent.Source)
	require.GreaterOrEqual(t, kgAgent.OntologyObjectsCount, 5)
	for _, n := range kgAgent.Nodes {
		if n.ID == drvID {
			require.Equal(t, ontology.RedactedPlaceholder, n.Name)
		}
	}

	st, b = call(http.MethodGet, "/v1/supplier/knowledge-graph", nil, ownerTok)
	require.Equal(t, http.StatusOK, st)
	var kgOwner planning.KnowledgeGraph
	require.NoError(t, json.Unmarshal(b, &kgOwner))
	require.Equal(t, "unified_ontology", kgOwner.Source)
	var sawDriver bool
	for _, n := range kgOwner.Nodes {
		if n.ID == drvID {
			sawDriver = true
			require.Equal(t, "Confidential E2E Driver", n.Name)
		}
	}
	require.True(t, sawDriver)
}

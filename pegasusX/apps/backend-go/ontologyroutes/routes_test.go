package ontologyroutes

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/ontology"
	"github.com/stretchr/testify/require"
)

type stubRebuildSource struct {
	objs []*ontology.OntologyObject
}

func (s *stubRebuildSource) LoadDomainSnapshot(_ context.Context, cellID, supplierID string) ([]*ontology.OntologyObject, []*ontology.OntologyGraphEdge, []string, error) {
	var out []*ontology.OntologyObject
	for _, o := range s.objs {
		if supplierID == "" || o.SupplierID == supplierID {
			cp := o.Clone()
			cp.CellID = cellID
			out = append(out, cp)
		}
	}
	return out, nil, []string{"Orders"}, nil
}

func (s *stubRebuildSource) ListActiveSupplierIDs(_ context.Context) ([]string, error) {
	return []string{"sup-1"}, nil
}

func newTestRouter(t *testing.T) (chi.Router, *ontology.MemoryRepository, *ontology.Service) {
	t.Helper()
	repo := ontology.NewMemoryRepository()
	svc := ontology.NewService(repo, nil).WithRebuildSource(&stubRebuildSource{
		objs: []*ontology.OntologyObject{
			{
				CellID:     ontology.DefaultCellID,
				ObjectType: ontology.TypeWarehouse,
				ObjectID:   "wh-rebuilt",
				SupplierID: "sup-1",
				Properties: map[string]any{"warehouse_id": "wh-rebuilt", "name": "Rebuilt Hub"},
			},
		},
	})
	_, err := svc.UpsertDirect(context.Background(), []*ontology.OntologyObject{
		{
			CellID:           ontology.DefaultCellID,
			ObjectType:       ontology.TypeOrder,
			ObjectID:         "ord-1",
			SupplierID:       "sup-1",
			Version:          1,
			SecurityMarkings: []string{ontology.MarkingFinanceOnly},
			Properties: map[string]any{
				"order_id":    "ord-1",
				"retailer_id": "shop-1",
				"status":      "CONFIRMED",
				"total_minor": int64(350000),
				"currency":    "UZS",
			},
		},
		{
			CellID:           ontology.DefaultCellID,
			ObjectType:       ontology.TypeShop,
			ObjectID:         "shop-1",
			SupplierID:       "sup-1",
			Version:          1,
			SecurityMarkings: []string{ontology.MarkingPII},
			Properties: map[string]any{
				"retailer_id": "shop-1",
				"name":        "Shop One",
				"phone":       "+998901112233",
			},
		},
	}, []*ontology.OntologyGraphEdge{
		{
			CellID:           ontology.DefaultCellID,
			SourceObjectType: ontology.TypeShop,
			SourceObjectID:   "shop-1",
			LinkType:         ontology.LinkOrdered,
			TargetObjectType: ontology.TypeOrder,
			TargetObjectID:   "ord-1",
			SupplierID:       "sup-1",
		},
	}, ontology.UpsertOptions{EmitOutbox: false})
	require.NoError(t, err)

	r := chi.NewRouter()
	return r, repo, svc
}

func doWithClaims(r chi.Router, method, path, body string, claims *auth.Claims) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if claims != nil {
		ctx := auth.WithClaims(req.Context(), *claims)
		req = req.WithContext(ctx)
	}
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec
}

func TestOntologyRoutesRBACRedactionAndTenantScope(t *testing.T) {
	r, _, svc := newTestRouter(t)
	RegisterRoutes(r, Deps{Service: svc})

	// 1. Unauthenticated caller -> 401
	require.Equal(t, http.StatusUnauthorized, doWithClaims(r, http.MethodGet, "/v1/ontology/objects", "", nil).Code)
	require.Equal(t, http.StatusUnauthorized, doWithClaims(r, http.MethodPost, "/v1/ontology/rebuild", `{}`, nil).Code)

	// 2. Retailer cannot access supplier ontology endpoints -> 403
	retailer := &auth.Claims{Subject: "ret-1", Role: auth.RoleRetailer}
	require.Equal(t, http.StatusForbidden, doWithClaims(r, http.MethodGet, "/v1/ontology/types", "", retailer).Code)

	// 3. AGENT / AUDITOR gets masked finance/PII values on reads, and 403 on rebuild
	agent := &auth.Claims{
		Subject:      "usr-agent",
		Role:         auth.RoleAdmin,
		SupplierID:   "sup-1",
		SupplierRole: auth.SupplierRoleAgent,
		ActorType:    auth.ActorTypeAgent,
	}
	typesRec := doWithClaims(r, http.MethodGet, "/v1/ontology/types", "", agent)
	require.Equal(t, http.StatusOK, typesRec.Code)
	var cat ontology.TypesCatalog
	require.NoError(t, json.Unmarshal(typesRec.Body.Bytes(), &cat))
	require.Len(t, cat.Types, 17)
	require.Len(t, cat.LinkTypes, 7)

	listRec := doWithClaims(r, http.MethodGet, "/v1/ontology/objects?type=Order", "", agent)
	require.Equal(t, http.StatusOK, listRec.Code)
	var listRes ontology.ListObjectsResult
	require.NoError(t, json.Unmarshal(listRec.Body.Bytes(), &listRes))
	require.True(t, listRes.FinanceMasked)
	require.Len(t, listRes.Objects, 1)
	require.Equal(t, ontology.RedactedPlaceholder, listRes.Objects[0].Properties["total_minor"])

	neighRec := doWithClaims(r, http.MethodGet, "/v1/ontology/objects/Order/ord-1/neighbors?depth=2", "", agent)
	require.Equal(t, http.StatusOK, neighRec.Code)
	var neighRes ontology.NeighborResult
	require.NoError(t, json.Unmarshal(neighRec.Body.Bytes(), &neighRes))
	require.True(t, neighRes.PIIMasked)
	require.Equal(t, 1, neighRes.TotalNodes)
	require.Equal(t, ontology.RedactedPlaceholder, neighRes.Nodes[0].Object.Properties["phone"])

	require.Equal(t, http.StatusForbidden, doWithClaims(r, http.MethodPost, "/v1/ontology/rebuild", `{}`, agent).Code)

	// 4. Cross-tenant query denied even for OWNER
	owner := &auth.Claims{
		Subject:      "usr-owner",
		Role:         auth.RoleAdmin,
		SupplierID:   "sup-1",
		SupplierRole: auth.SupplierRoleOwner,
	}
	require.Equal(t, http.StatusForbidden, doWithClaims(r, http.MethodGet, "/v1/ontology/objects?supplier_id=sup-2", "", owner).Code)

	// 5. OWNER receives unmasked values and can trigger rebuild
	ownerList := doWithClaims(r, http.MethodGet, "/v1/ontology/objects?type=Order", "", owner)
	require.Equal(t, http.StatusOK, ownerList.Code)
	var ownerRes ontology.ListObjectsResult
	require.NoError(t, json.Unmarshal(ownerList.Body.Bytes(), &ownerRes))
	require.False(t, ownerRes.FinanceMasked)
	require.EqualValues(t, 350000, ownerRes.Objects[0].Properties["total_minor"])

	rebuildRec := doWithClaims(r, http.MethodPost, "/v1/ontology/rebuild", `{}`, owner)
	require.Equal(t, http.StatusOK, rebuildRec.Code)
	var rbRes ontology.RebuildResult
	require.NoError(t, json.Unmarshal(rebuildRec.Body.Bytes(), &rbRes))
	require.Equal(t, "COMPLETED", rbRes.Status)
	require.Equal(t, 1, rbRes.ObjectsUpserted)

	// 6. Unknown type -> 400; missing object -> 404
	require.Equal(t, http.StatusBadRequest, doWithClaims(r, http.MethodGet, "/v1/ontology/objects?type=Invalid", "", owner).Code)
	require.Equal(t, http.StatusNotFound, doWithClaims(r, http.MethodGet, "/v1/ontology/objects/Order/does-not-exist/neighbors", "", owner).Code)
}

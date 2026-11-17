package supplier

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/ontology"
	"github.com/pegasusx/pegasusx/apps/backend-go/planning"
	"github.com/stretchr/testify/require"
)

func TestHandleKnowledgeGraphUnifiedOntology(t *testing.T) {
	ctx := context.Background()
	repo := ontology.NewMemoryRepository()
	ontSvc := ontology.NewService(repo, nil)
	_, err := ontSvc.UpsertDirect(ctx, []*ontology.OntologyObject{
		{
			CellID:     ontology.DefaultCellID,
			ObjectType: ontology.TypeSupplier,
			ObjectID:   "sup-portal-1",
			SupplierID: "sup-portal-1",
			Properties: map[string]any{"supplier_id": "sup-portal-1", "name": "Portal Supplier"},
		},
		{
			CellID:     ontology.DefaultCellID,
			ObjectType: ontology.TypeWarehouse,
			ObjectID:   "wh-portal-1",
			SupplierID: "sup-portal-1",
			Properties: map[string]any{"warehouse_id": "wh-portal-1", "name": "Portal Depot"},
		},
	}, []*ontology.OntologyGraphEdge{
		{
			CellID:           ontology.DefaultCellID,
			SourceObjectType: ontology.TypeSupplier,
			SourceObjectID:   "sup-portal-1",
			LinkType:         ontology.LinkSupplies,
			TargetObjectType: ontology.TypeWarehouse,
			TargetObjectID:   "wh-portal-1",
			SupplierID:       "sup-portal-1",
		},
	}, ontology.UpsertOptions{EmitOutbox: false})
	require.NoError(t, err)

	svc := NewService(ServiceConfig{SupplierID: "sup-portal-1"})
	svc.SetUnifiedGraph(planning.AdaptOntologyService(ontSvc, ontology.DefaultCellID))

	req := httptest.NewRequest(http.MethodGet, "/v1/supplier/knowledge-graph", nil)
	req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{
		Subject:      "usr-1",
		Role:         auth.RoleAdmin,
		SupplierID:   "sup-portal-1",
		SupplierRole: auth.SupplierRoleOwner,
	}))
	rec := httptest.NewRecorder()
	svc.HandleKnowledgeGraph(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var kg planning.KnowledgeGraph
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &kg))
	require.Equal(t, "sup-portal-1", kg.SupplierID)
	require.Equal(t, "unified_ontology", kg.Source)
	require.Len(t, kg.Nodes, 2)
	require.Len(t, kg.Edges, 1)
}

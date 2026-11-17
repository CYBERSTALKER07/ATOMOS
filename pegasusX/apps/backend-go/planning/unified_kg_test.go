package planning

import (
	"context"
	"testing"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/ontology"
	"github.com/stretchr/testify/require"
)

func TestGetKnowledgeGraphUnifiedAndRedacted(t *testing.T) {
	ctx := context.Background()
	repo := ontology.NewMemoryRepository()
	ontSvc := ontology.NewService(repo, nil)
	_, err := ontSvc.UpsertDirect(ctx, []*ontology.OntologyObject{
		{
			CellID:     ontology.DefaultCellID,
			ObjectType: ontology.TypeSupplier,
			ObjectID:   "sup-ekg-1",
			SupplierID: "sup-ekg-1",
			Properties: map[string]any{"supplier_id": "sup-ekg-1", "name": "Samarkand Dairy"},
		},
		{
			CellID:     ontology.DefaultCellID,
			ObjectType: ontology.TypeWarehouse,
			ObjectID:   "wh-ekg-1",
			SupplierID: "sup-ekg-1",
			Properties: map[string]any{"warehouse_id": "wh-ekg-1", "name": "Central Cold Hub"},
		},
		{
			CellID:           ontology.DefaultCellID,
			ObjectType:       ontology.TypeDriver,
			ObjectID:         "drv-ekg-1",
			SupplierID:       "sup-ekg-1",
			SecurityMarkings: []string{ontology.MarkingPII},
			Properties:       map[string]any{"driver_id": "drv-ekg-1", "name": "Confidential Driver", "phone": "+998901112233"},
		},
		{
			CellID:     ontology.DefaultCellID,
			ObjectType: ontology.TypeNasiyaInvoice,
			ObjectID:   "inv-ekg-1",
			SupplierID: "sup-ekg-1",
			Properties: map[string]any{
				"invoice_id":      "inv-ekg-1",
				"retailer_id":     "shop-1",
				"status":          "OPEN",
				"principal_minor": int64(500000),
				"balance_minor":   int64(250000),
				"currency":        "UZS",
			},
		},
	}, []*ontology.OntologyGraphEdge{
		{
			CellID:           ontology.DefaultCellID,
			SourceObjectType: ontology.TypeSupplier,
			SourceObjectID:   "sup-ekg-1",
			LinkType:         ontology.LinkSupplies,
			TargetObjectType: ontology.TypeWarehouse,
			TargetObjectID:   "wh-ekg-1",
			SupplierID:       "sup-ekg-1",
		},
	}, ontology.UpsertOptions{EmitOutbox: false})
	require.NoError(t, err)

	planSvc := NewService(nil).WithUnifiedGraph(AdaptOntologyService(ontSvc, ontology.DefaultCellID))

	kgOwner, err := planSvc.GetKnowledgeGraphForViewer(ctx, "sup-ekg-1", auth.SupplierRoleOwner)
	require.NoError(t, err)
	require.Equal(t, "sup-ekg-1", kgOwner.SupplierID)
	require.Equal(t, "unified_ontology", kgOwner.Source)
	require.Equal(t, 4, kgOwner.OntologyObjectsCount)
	require.Equal(t, 1, kgOwner.OntologyEdgesCount)
	require.Len(t, kgOwner.Nodes, 4)
	require.Len(t, kgOwner.Edges, 1)

	var foundDriverOwner bool
	for _, n := range kgOwner.Nodes {
		if n.ID == "drv-ekg-1" {
			foundDriverOwner = true
			require.Equal(t, "driver", n.Type)
			require.Equal(t, "Confidential Driver", n.Name)
		}
	}
	require.True(t, foundDriverOwner)

	kgAgent, err := planSvc.GetKnowledgeGraphForViewer(ctx, "sup-ekg-1", auth.SupplierRoleAgent)
	require.NoError(t, err)
	for _, n := range kgAgent.Nodes {
		if n.ID == "drv-ekg-1" {
			require.Equal(t, ontology.RedactedPlaceholder, n.Name)
		}
	}
}

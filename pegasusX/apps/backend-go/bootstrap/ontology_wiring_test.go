package bootstrap

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/ontology"
	"github.com/pegasusx/pegasusx/apps/backend-go/planning"
	"github.com/stretchr/testify/require"
)

func TestOntologyConsumerTopicsExcludesSelfIntelTopic(t *testing.T) {
	t.Setenv("KAFKA_TOPIC_CONSUME_DOMAIN", "true")
	topics := OntologyConsumerTopics()
	require.Contains(t, topics, events.TopicMain)
	require.Contains(t, topics, events.TopicOrders)
	require.Contains(t, topics, events.TopicDispatch)
	require.Contains(t, topics, events.TopicExceptions)
	require.Contains(t, topics, events.TopicLedger)
	require.Contains(t, topics, events.TopicConnectors)
	require.NotContains(t, topics, events.TopicIntel)

	t.Setenv("KAFKA_TOPIC_CONSUME_DOMAIN", "false")
	require.Equal(t, []string{events.TopicMain}, OntologyConsumerTopics())
}

func TestNewApp_WiresOntologyServiceConsumerAndUnifiedKG(t *testing.T) {
	app := setupStubAppConfig(t, RunModeAll)
	require.NotNil(t, app.Ontology)
	require.NotNil(t, app.OntologyEventConsumer)
	require.NotNil(t, app.SupplierService)

	_, err := app.Ontology.UpsertDirect(context.Background(), []*ontology.OntologyObject{
		{
			CellID:     ontology.DefaultCellID,
			ObjectType: ontology.TypeSupplier,
			ObjectID:   app.Supplier.SupplierID,
			SupplierID: app.Supplier.SupplierID,
			Properties: map[string]any{"supplier_id": app.Supplier.SupplierID, "name": "Boot Supplier"},
		},
		{
			CellID:     ontology.DefaultCellID,
			ObjectType: ontology.TypeWarehouse,
			ObjectID:   "wh-boot-1",
			SupplierID: app.Supplier.SupplierID,
			Properties: map[string]any{"warehouse_id": "wh-boot-1", "name": "Boot Depot"},
		},
	}, []*ontology.OntologyGraphEdge{
		{
			CellID:           ontology.DefaultCellID,
			SourceObjectType: ontology.TypeSupplier,
			SourceObjectID:   app.Supplier.SupplierID,
			LinkType:         ontology.LinkSupplies,
			TargetObjectType: ontology.TypeWarehouse,
			TargetObjectID:   "wh-boot-1",
			SupplierID:       app.Supplier.SupplierID,
		},
	}, ontology.UpsertOptions{EmitOutbox: false})
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/v1/supplier/knowledge-graph", nil)
	req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{
		Subject:      "boot-owner",
		Role:         auth.RoleAdmin,
		SupplierID:   app.Supplier.SupplierID,
		SupplierRole: auth.SupplierRoleOwner,
	}))
	rec := httptest.NewRecorder()
	app.SupplierService.HandleKnowledgeGraph(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	var kg planning.KnowledgeGraph
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &kg))
	require.Equal(t, "unified_ontology", kg.Source)
	require.Equal(t, 2, kg.OntologyObjectsCount)
	require.Equal(t, 1, kg.OntologyEdgesCount)
	require.WithinDuration(t, time.Now(), time.Now(), time.Second)
}

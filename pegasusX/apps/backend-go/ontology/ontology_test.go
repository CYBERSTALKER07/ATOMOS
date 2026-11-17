package ontology

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	kafka "github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/require"
)

type stubRebuildSource struct {
	objs   []*OntologyObject
	edges  []*OntologyGraphEdge
	tables []string
}

func (s *stubRebuildSource) LoadDomainSnapshot(_ context.Context, cellID, supplierID string) ([]*OntologyObject, []*OntologyGraphEdge, []string, error) {
	var outObjs []*OntologyObject
	for _, o := range s.objs {
		if supplierID == "" || o.SupplierID == supplierID {
			cp := o.Clone()
			cp.CellID = cellID
			outObjs = append(outObjs, cp)
		}
	}
	var outEdges []*OntologyGraphEdge
	for _, e := range s.edges {
		if supplierID == "" || e.SupplierID == supplierID {
			cp := *e
			cp.CellID = cellID
			outEdges = append(outEdges, &cp)
		}
	}
	return outObjs, outEdges, append([]string(nil), s.tables...), nil
}

func (s *stubRebuildSource) ListActiveSupplierIDs(_ context.Context) ([]string, error) {
	seen := map[string]bool{}
	var ids []string
	for _, o := range s.objs {
		if o.SupplierID != "" && !seen[o.SupplierID] {
			seen[o.SupplierID] = true
			ids = append(ids, o.SupplierID)
		}
	}
	return ids, nil
}

func TestRegistryHasAll17TypesAnd7Links(t *testing.T) {
	cat := ListTypes()
	require.Len(t, cat.Types, 17)
	require.Len(t, cat.LinkTypes, 7)
	require.Equal(t, SchemaVersion, cat.Version)

	expectedTypes := []string{
		TypeSupplier, TypeLegalEntity, TypeWarehouse, TypeFactory, TypeSKU, TypeLot,
		TypeShop, TypeOrder, TypeOrderLine, TypeManifest, TypeVehicle, TypeDriver,
		TypeNasiyaInvoice, TypeCreditLine, TypePayment, TypeFiscalDoc, TypeProposal,
	}
	for _, typ := range expectedTypes {
		desc, ok := LookupType(typ)
		require.True(t, ok, "missing type %s", typ)
		require.Equal(t, "v1", desc.Version)
	}
	for _, lt := range AllLinkTypes() {
		require.True(t, IsValidLinkType(lt), "missing link %s", lt)
	}
	require.False(t, IsValidLinkType("bogus_link"))
}

func TestValidateObjectRejectsFloatsAndMissingFields(t *testing.T) {
	err := ValidateObject(&OntologyObject{
		CellID:     "cell-uz",
		ObjectType: TypeOrder,
		ObjectID:   "ord-float",
		SupplierID: "sup-1",
		Properties: map[string]any{
			"order_id":    "ord-float",
			"retailer_id": "ret-1",
			"status":      "CREATED",
			"total_minor": 125.50, // float must fail!
			"currency":    "UZS",
		},
	})
	require.Error(t, err)
	require.True(t, errors.Is(err, ErrFloatForbidden), "expected ErrFloatForbidden, got %v", err)

	_, decErr := DecodeJSONMapNoFloats([]byte(`{"price_minor": 99.95}`))
	require.ErrorIs(t, decErr, ErrFloatForbidden)

	errMissing := ValidateObject(&OntologyObject{
		CellID:     "cell-uz",
		ObjectType: TypeOrder,
		ObjectID:   "ord-missing",
		SupplierID: "sup-1",
		Properties: map[string]any{
			"order_id": "ord-missing",
			"status":   "CREATED",
		},
	})
	require.ErrorIs(t, errMissing, ErrSchemaValidation)

	errOK := ValidateObject(&OntologyObject{
		CellID:     "cell-uz",
		ObjectType: TypeOrder,
		ObjectID:   "ord-ok",
		SupplierID: "sup-1",
		Properties: map[string]any{
			"order_id":    "ord-ok",
			"retailer_id": "ret-1",
			"status":      "CREATED",
			"total_minor": int64(250000),
			"currency":    "UZS",
		},
	})
	require.NoError(t, errOK)
}

func TestMonotonicVersionIgnoresOutOfOrderEvents(t *testing.T) {
	ctx := context.Background()
	repo := NewMemoryRepository()
	svc := NewService(repo, slog.New(slog.NewTextHandler(io.Discard, nil)))
	consumer := NewEventConsumer(svc, slog.New(slog.NewTextHandler(io.Discard, nil)))

	// 1. Incoming v5 event arrives first.
	v5 := []byte(`{"type":"ORDER_STATUS_CHANGED","version":5,"order_id":"ord-1","supplier_id":"sup-1","retailer_id":"shop-1","status":"DELIVERED","total_minor":500000,"currency":"UZS"}`)
	require.NoError(t, consumer.HandleEvent(ctx, kafka.Message{Value: v5}))

	obj, _, err := svc.GetObject(ctx, DefaultCellID, TypeOrder, "ord-1", auth.SupplierRoleOwner)
	require.NoError(t, err)
	require.Equal(t, int64(5), obj.Version)
	require.Equal(t, "DELIVERED", obj.Properties["status"])
	require.Len(t, repo.OutboxRecords(), 1)
	require.Equal(t, events.TopicIntel, repo.OutboxRecords()[0].Topic)
	require.Equal(t, events.EventOntologyObjectChanged, repo.OutboxRecords()[0].EventType)

	// 2. Stale out-of-order v3 event arrives later -> must be ignored cleanly!
	v3 := []byte(`{"type":"ORDER_CREATED","version":3,"order_id":"ord-1","supplier_id":"sup-1","retailer_id":"shop-1","status":"CREATED","total_minor":450000,"currency":"UZS"}`)
	require.NoError(t, consumer.HandleEvent(ctx, kafka.Message{Value: v3}))

	afterStale, _, err := svc.GetObject(ctx, DefaultCellID, TypeOrder, "ord-1", auth.SupplierRoleOwner)
	require.NoError(t, err)
	require.Equal(t, int64(5), afterStale.Version)
	require.Equal(t, "DELIVERED", afterStale.Properties["status"])
	require.Equal(t, int64(500000), afterStale.Properties["total_minor"])
	// Outbox must NOT have grown when stale v3 was skipped.
	require.Len(t, repo.OutboxRecords(), 1)

	// 3. Duplicate v5 is also ignored cleanly; v6 advances cleanly.
	require.NoError(t, consumer.HandleEvent(ctx, kafka.Message{Value: v5}))
	require.Len(t, repo.OutboxRecords(), 1)

	v6 := []byte(`{"type":"ORDER_FINALIZED","version":6,"order_id":"ord-1","supplier_id":"sup-1","retailer_id":"shop-1","status":"COMPLETED","total_minor":500000,"currency":"UZS"}`)
	require.NoError(t, consumer.HandleEvent(ctx, kafka.Message{Value: v6}))
	afterV6, _, err := svc.GetObject(ctx, DefaultCellID, TypeOrder, "ord-1", auth.SupplierRoleOwner)
	require.NoError(t, err)
	require.Equal(t, int64(6), afterV6.Version)
	require.Equal(t, "COMPLETED", afterV6.Properties["status"])
	require.Len(t, repo.OutboxRecords(), 2)
}

func TestKafkaDomainEventProjectorAllDomainsAndSelfEchoSkip(t *testing.T) {
	ctx := context.Background()
	repo := NewMemoryRepository()
	svc := NewService(repo, slog.New(slog.NewTextHandler(io.Discard, nil)))
	consumer := NewEventConsumer(svc, slog.New(slog.NewTextHandler(io.Discard, nil)))

	msgs := []struct {
		topic string
		body  string
	}{
		{
			topic: events.TopicMain,
			body:  `{"type":"SUPPLIER_CREATED","version":1,"supplier_id":"sup-1","name":"Tashkent Foods","legal_name":"Tashkent Foods LLC","phone":"+998901112233","is_registered":true}`,
		},
		{
			topic: events.TopicMain,
			body:  `{"type":"RETAILER_REGISTERED","version":1,"supplier_id":"sup-1","retailer_id":"shop-1","name":"Safia Market","phone":"+998909998877","lat":41.3111,"lng":69.2797}`,
		},
		{
			topic: events.TopicMain,
			body:  `{"type":"WAREHOUSE_CREATED","version":1,"supplier_id":"sup-1","warehouse_id":"wh-1","name":"Sergeli Hub"}`,
		},
		{
			topic: events.TopicMain,
			body:  `{"type":"PRODUCT_CREATED","version":1,"supplier_id":"sup-1","warehouse_id":"wh-1","product_id":"sku-1","name":"Cola 1L","price_minor":12000,"currency":"UZS"}`,
		},
		{
			topic: events.TopicOrders,
			body:  `{"type":"order.created","version":1,"supplier_id":"sup-1","retailer_id":"shop-1","warehouse_id":"wh-1","order_id":"ord-10","status":"CONFIRMED","total_minor":120000,"currency":"UZS","lat":41.31,"lng":69.28,"line_items":[{"product_id":"sku-1","sku":"COLA-1L","name":"Cola 1L","quantity":10,"unit_price_minor":12000}]}`,
		},
		{
			topic: events.TopicDispatch,
			body:  `{"type":"manifest.dispatched","version":1,"supplier_id":"sup-1","warehouse_id":"wh-1","manifest_id":"man-1","vehicle_id":"veh-1","driver_id":"drv-1","order_id":"ord-10","state":"DISPATCHED"}`,
		},
		{
			topic: events.TopicMain,
			body:  `{"type":"LOT_RECALL_INITIATED","version":1,"supplier_id":"sup-1","warehouse_id":"wh-1","product_id":"sku-1","lot_id":"lot-bad","order_id":"ord-10","recall_reason":"THERMAL_SPIKE"}`,
		},
		{
			topic: events.TopicMain,
			body:  `{"type":"credit.updated","version":1,"supplier_id":"sup-1","retailer_id":"shop-1","credit_line_id":"cl-1","credit_limit_minor":5000000,"current_balance_minor":120000}`,
		},
		{
			topic: events.TopicMain,
			body:  `{"type":"AR_INVOICE_OPENED","version":1,"supplier_id":"sup-1","retailer_id":"shop-1","order_id":"ord-10","invoice_id":"inv-1","principal_minor":120000,"balance_minor":120000}`,
		},
		{
			topic: events.TopicMain,
			body:  `{"type":"PAYMENT_CLEARED","version":1,"supplier_id":"sup-1","retailer_id":"shop-1","order_id":"ord-10","invoice_id":"inv-1","payment_id":"pay-1","amount_minor":120000,"gateway":"payme"}`,
		},
		{
			topic: events.TopicMain,
			body:  `{"type":"FISCAL_RECEIPT_SUCCEEDED","version":1,"supplier_id":"sup-1","retailer_id":"shop-1","order_id":"ord-10","attempt_id":"fis-1","fiscal_receipt_id":"rcpt-1","provider":"SOLIQ_OFD","amount_minor":120000}`,
		},
		{
			topic: events.TopicLedger,
			body:  `{"type":"gl.entry.posted","version":1,"entity_id":"sup-1","entry_id":"je-1","period_id":"2026-11","source_type":"PAYMENT","source_id":"pay-gl-1","currency":"UZS","total_debit_minor":50000,"total_credit_minor":50000}`,
		},
		{
			topic: events.TopicExceptions,
			body:  `{"type":"AML_EXPOSURE_UPDATED","version":1,"supplier_id":"sup-1","legal_entity_id":"le-aml-1","legal_name":"Alpha Trade LLC","tin":"300112233","daily_cash_accumulated_minor":120000000,"daily_cash_ceiling_minor":100000000}`,
		},
		{
			topic: events.TopicMain,
			body:  `{"type":"NEGOTIATION_PROPOSED","version":1,"supplier_id":"sup-1","order_id":"ord-10","proposal_id":"prop-1","proposed_price_minor":115000}`,
		},
		// Self-echo event on TopicIntel must be ignored cleanly (zero outbox amplification).
		{
			topic: events.TopicIntel,
			body:  `{"type":"ontology.object.changed","cell_id":"cell-uz","object_type":"Order","object_id":"ord-10","supplier_id":"sup-1","version":1}`,
		},
	}

	for _, tc := range msgs {
		require.NoError(t, consumer.HandleEvent(ctx, kafka.Message{
			Topic: tc.topic,
			Value: []byte(tc.body),
		}), "event failed: %s", tc.body)
	}

	// Verify recall quarantine marking on Lot
	lot, _, err := svc.GetObject(ctx, DefaultCellID, TypeLot, "lot-bad", auth.SupplierRoleOwner)
	require.NoError(t, err)
	require.Contains(t, lot.SecurityMarkings, MarkingLotRecallQuarantine)

	// Verify AML investigation marking on LegalEntity
	le, _, err := svc.GetObject(ctx, DefaultCellID, TypeLegalEntity, "le-aml-1", auth.SupplierRoleOwner)
	require.NoError(t, err)
	require.Contains(t, le.SecurityMarkings, MarkingAMLInvestigation)
	require.Contains(t, le.SecurityMarkings, MarkingFinanceOnly)
	require.Contains(t, le.SecurityMarkings, MarkingPII)

	// Verify float rejection on Kafka money field
	floatMsg := kafka.Message{
		Topic: events.TopicOrders,
		Value: []byte(`{"type":"order.created","version":1,"order_id":"ord-bad","supplier_id":"sup-1","retailer_id":"shop-1","total_minor":12.34}`),
	}
	err = consumer.HandleEvent(ctx, floatMsg)
	require.ErrorIs(t, err, ErrFloatForbidden)

	// Verify Outbox events recorded onto TopicIntel
	outbox := repo.OutboxRecords()
	require.NotEmpty(t, outbox)
	for _, rec := range outbox {
		require.Equal(t, events.TopicIntel, rec.Topic)
		require.Equal(t, events.EventOntologyObjectChanged, rec.EventType)
		var ev events.OntologyObjectChangedEvent
		require.NoError(t, json.Unmarshal([]byte(rec.PayloadJSON), &ev))
		require.NotEmpty(t, ev.ObjectType)
		require.NotEmpty(t, ev.ObjectID)
	}
}

func TestViewerRedactionSecurityMarkings(t *testing.T) {
	ctx := context.Background()
	repo := NewMemoryRepository()
	svc := NewService(repo, nil)

	_, err := svc.UpsertDirect(ctx, []*OntologyObject{
		{
			CellID:           DefaultCellID,
			ObjectType:       TypeOrder,
			ObjectID:         "ord-sec-1",
			SupplierID:       "sup-1",
			Version:          1,
			SecurityMarkings: []string{MarkingFinanceOnly},
			Properties: map[string]any{
				"order_id":    "ord-sec-1",
				"retailer_id": "shop-sec-1",
				"status":      "CONFIRMED",
				"total_minor": int64(890000),
				"currency":    "UZS",
			},
		},
		{
			CellID:           DefaultCellID,
			ObjectType:       TypeShop,
			ObjectID:         "shop-sec-1",
			SupplierID:       "sup-1",
			Version:          1,
			SecurityMarkings: []string{MarkingPII},
			Properties: map[string]any{
				"retailer_id":      "shop-sec-1",
				"name":             "Safia Market",
				"phone":            "+998901234567",
				"delivery_address": "Chilonzor 12",
			},
		},
		{
			CellID:           DefaultCellID,
			ObjectType:       TypeDriver,
			ObjectID:         "drv-sec-1",
			SupplierID:       "sup-1",
			Version:          1,
			SecurityMarkings: []string{MarkingPII},
			Properties: map[string]any{
				"driver_id": "drv-sec-1",
				"name":      "Azizbek Karimov",
				"phone":     "+998907654321",
			},
		},
	}, nil, UpsertOptions{EmitOutbox: false})
	require.NoError(t, err)

	// Finance/Owner sees unmasked figures & PII
	adminRes, err := svc.ListObjects(ctx, ListObjectsFilter{
		CellID:     DefaultCellID,
		SupplierID: "sup-1",
		ViewerRole: auth.SupplierRoleFinanceManager,
	})
	require.NoError(t, err)
	require.False(t, adminRes.PIIMasked)
	require.False(t, adminRes.FinanceMasked)

	// Auditor / Agent sees [REDACTED] on finance & PII fields
	agentRes, err := svc.ListObjects(ctx, ListObjectsFilter{
		CellID:     DefaultCellID,
		SupplierID: "sup-1",
		ViewerRole: auth.SupplierRoleAgent,
	})
	require.NoError(t, err)
	require.True(t, agentRes.PIIMasked)
	require.True(t, agentRes.FinanceMasked)

	byType := map[string]*OntologyObject{}
	for _, o := range agentRes.Objects {
		byType[o.ObjectType] = o
	}
	require.Equal(t, RedactedPlaceholder, byType[TypeOrder].Properties["total_minor"])
	require.Equal(t, "CONFIRMED", byType[TypeOrder].Properties["status"])
	require.Equal(t, RedactedPlaceholder, byType[TypeShop].Properties["phone"])
	require.Equal(t, RedactedPlaceholder, byType[TypeShop].Properties["delivery_address"])
	require.Equal(t, "Safia Market", byType[TypeShop].Properties["name"])
	require.Equal(t, RedactedPlaceholder, byType[TypeDriver].Properties["name"])
	require.Equal(t, RedactedPlaceholder, byType[TypeDriver].Properties["phone"])
}

func TestGetNeighborsMultiHopAndBatchChunking(t *testing.T) {
	ctx := context.Background()
	repo := NewMemoryRepository()
	svc := NewService(repo, nil)

	objs := []*OntologyObject{
		{CellID: DefaultCellID, ObjectType: TypeWarehouse, ObjectID: "wh-1", SupplierID: "sup-1", Properties: map[string]any{"warehouse_id": "wh-1", "name": "Hub"}},
		{CellID: DefaultCellID, ObjectType: TypeLot, ObjectID: "lot-1", SupplierID: "sup-1", Properties: map[string]any{"lot_id": "lot-1", "warehouse_id": "wh-1", "product_id": "sku-1", "status": "ACTIVE"}},
		{CellID: DefaultCellID, ObjectType: TypeOrder, ObjectID: "ord-1", SupplierID: "sup-1", Properties: map[string]any{"order_id": "ord-1", "retailer_id": "shop-1", "status": "DELIVERED", "total_minor": int64(50000), "currency": "UZS"}},
		{CellID: DefaultCellID, ObjectType: TypeShop, ObjectID: "shop-1", SupplierID: "sup-1", Properties: map[string]any{"retailer_id": "shop-1", "name": "Shop One", "phone": "+998901111111"}},
	}
	edges := []*OntologyGraphEdge{
		{CellID: DefaultCellID, SourceObjectType: TypeWarehouse, SourceObjectID: "wh-1", LinkType: LinkStocks, TargetObjectType: TypeLot, TargetObjectID: "lot-1", SupplierID: "sup-1"},
		{CellID: DefaultCellID, SourceObjectType: TypeLot, SourceObjectID: "lot-1", LinkType: LinkOrdered, TargetObjectType: TypeOrder, TargetObjectID: "ord-1", SupplierID: "sup-1"},
		{CellID: DefaultCellID, SourceObjectType: TypeShop, SourceObjectID: "shop-1", LinkType: LinkOrdered, TargetObjectType: TypeOrder, TargetObjectID: "ord-1", SupplierID: "sup-1"},
	}
	_, err := svc.UpsertDirect(ctx, objs, edges, UpsertOptions{EmitOutbox: false})
	require.NoError(t, err)

	res, err := svc.GetNeighbors(ctx, DefaultCellID, "sup-1", TypeWarehouse, "wh-1", 3, nil, auth.SupplierRoleAuditor)
	require.NoError(t, err)
	require.Equal(t, 3, res.TotalNodes)
	require.Equal(t, 3, res.TotalEdges)
	require.True(t, res.PIIMasked)

	// Cross-tenant isolation check
	_, err = svc.GetNeighbors(ctx, DefaultCellID, "sup-foreign", TypeWarehouse, "wh-1", 2, nil, auth.SupplierRoleOwner)
	require.ErrorIs(t, err, ErrObjectNotFound)

	// Batch > 900 mutations chunking verification (1,350 objects + 1,050 edges)
	manyObjs := make([]*OntologyObject, 0, 1350)
	manyEdges := make([]*OntologyGraphEdge, 0, 1050)
	for i := 0; i < 1350; i++ {
		id := "wh-bulk-" + string(rune('a'+(i%26))) + "-" + time.Now().Add(time.Duration(i)*time.Nanosecond).Format("150405.000000000")
		if i < 10 {
			id = "wh-bulk-" + string(rune('0'+i))
		} else {
			id = "wh-bulk-" + string([]byte{
				byte('a' + (i/676)%26),
				byte('a' + (i/26)%26),
				byte('a' + i%26),
			})
		}
		manyObjs = append(manyObjs, &OntologyObject{
			CellID:     DefaultCellID,
			ObjectType: TypeWarehouse,
			ObjectID:   id,
			SupplierID: "sup-bulk",
			Properties: map[string]any{"warehouse_id": id, "name": "Bulk " + id},
		})
		if i > 0 && i <= 1050 {
			manyEdges = append(manyEdges, &OntologyGraphEdge{
				CellID:           DefaultCellID,
				SourceObjectType: TypeWarehouse,
				SourceObjectID:   manyObjs[0].ObjectID,
				LinkType:         LinkSupplies,
				TargetObjectType: TypeWarehouse,
				TargetObjectID:   id,
				SupplierID:       "sup-bulk",
			})
		}
	}
	stats, err := svc.UpsertDirect(ctx, manyObjs, manyEdges, UpsertOptions{EmitOutbox: true})
	require.NoError(t, err)
	require.Equal(t, 1350, stats.ObjectsUpserted)
	require.Equal(t, 1050, stats.EdgesUpserted)
	// 1350/450 = 3 object batches + ceil(1050/900) = 2 edge batches = 5 batches!
	require.GreaterOrEqual(t, stats.BatchesCommitted, 5)
}

func TestRebuildFromSourceAndKnowledgeGraphUnified(t *testing.T) {
	ctx := context.Background()
	repo := NewMemoryRepository()
	src := &stubRebuildSource{
		tables: []string{"Suppliers", "Warehouses", "Drivers"},
		objs: []*OntologyObject{
			{CellID: DefaultCellID, ObjectType: TypeSupplier, ObjectID: "sup-1", SupplierID: "sup-1", Properties: map[string]any{"supplier_id": "sup-1", "name": "Sup One"}},
			{CellID: DefaultCellID, ObjectType: TypeWarehouse, ObjectID: "wh-1", SupplierID: "sup-1", Properties: map[string]any{"warehouse_id": "wh-1", "name": "Hub 1"}},
			{CellID: DefaultCellID, ObjectType: TypeDriver, ObjectID: "drv-1", SupplierID: "sup-1", Properties: map[string]any{"driver_id": "drv-1", "name": "Secret Driver", "phone": "+998901230000"}},
		},
		edges: []*OntologyGraphEdge{
			{CellID: DefaultCellID, SourceObjectType: TypeSupplier, SourceObjectID: "sup-1", LinkType: LinkSupplies, TargetObjectType: TypeWarehouse, TargetObjectID: "wh-1", SupplierID: "sup-1"},
		},
	}
	svc := NewService(repo, nil).WithRebuildSource(src)

	first, err := svc.RebuildFromSpanner(ctx, DefaultCellID, "sup-1")
	require.NoError(t, err)
	require.Equal(t, 3, first.ObjectsUpserted)
	require.Equal(t, 1, first.EdgesUpserted)
	require.Equal(t, []string{"Suppliers", "Warehouses", "Drivers"}, first.TablesScanned)

	// Nightly idempotency check: running rebuild again with zero drift skips redundant version/outbox writes
	second, err := svc.RebuildFromSpanner(ctx, DefaultCellID, "sup-1")
	require.NoError(t, err)
	require.Equal(t, 0, second.ObjectsUpserted)
	require.Len(t, repo.OutboxRecords(), 3)

	kgAdmin, err := svc.GetKnowledgeGraphUnified(ctx, DefaultCellID, "sup-1", auth.SupplierRoleOwner)
	require.NoError(t, err)
	require.Equal(t, "unified_ontology", kgAdmin.Source)
	require.Equal(t, 3, kgAdmin.OntologyObjectsCount)
	require.Equal(t, 1, kgAdmin.OntologyEdgesCount)

	kgAgent, err := svc.GetKnowledgeGraphUnified(ctx, DefaultCellID, "sup-1", auth.SupplierRoleAgent)
	require.NoError(t, err)
	for _, n := range kgAgent.Nodes {
		if n.Type == "driver" {
			require.Equal(t, RedactedPlaceholder, n.Name)
		}
	}
}

package ontology

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strings"
	"testing"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	pegasuskafka "github.com/pegasusx/pegasusx/apps/backend-go/kafka"
	kgo "github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/require"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
)

func openOntologySpannerClient(t *testing.T) *spanner.Client {
	t.Helper()
	requireSpanner := strings.TrimSpace(os.Getenv("PARITY_REQUIRE_SPANNER")) == "1"
	host := strings.TrimSpace(os.Getenv("SPANNER_EMULATOR_HOST"))
	if host == "" {
		if requireSpanner {
			t.Fatal("PARITY_REQUIRE_SPANNER=1 but SPANNER_EMULATOR_HOST is unset")
		}
		t.Skip("SPANNER_EMULATOR_HOST not set; skipping live Spanner ontology test")
	}
	envOr := func(k, def string) string {
		if v := strings.TrimSpace(os.Getenv(k)); v != "" {
			return v
		}
		return def
	}
	dbPath := fmt.Sprintf("projects/%s/instances/%s/databases/%s",
		envOr("SPANNER_PROJECT", "pegasusx-local"),
		envOr("SPANNER_INSTANCE", "pegasusx-instance"),
		envOr("SPANNER_DATABASE", "pegasusx-db"))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	client, err := spanner.NewClient(ctx, dbPath,
		option.WithEndpoint(host),
		option.WithoutAuthentication(),
		option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())),
	)
	if err != nil {
		if requireSpanner {
			t.Fatalf("spanner connect: %v", err)
		}
		t.Skipf("spanner unavailable: %v", err)
	}
	ro := client.ReadOnlyTransaction().WithTimestampBound(spanner.StrongRead())
	_, probeErr := ro.ReadRow(ctx, "OntologyObjects", spanner.Key{"__probe__", TypeOrder, "__probe__"}, []string{"ObjectId"})
	ro.Close()
	if probeErr != nil && spanner.ErrCode(probeErr) != codes.NotFound {
		client.Close()
		if requireSpanner {
			t.Fatalf("OntologyObjects unavailable: %v", probeErr)
		}
		t.Skipf("OntologyObjects unavailable: %v", probeErr)
	}
	t.Cleanup(client.Close)
	return client
}

func cleanupOntologyCellRows(t *testing.T, client *spanner.Client, cellID string, outboxIDs []string) {
	t.Helper()
	t.Cleanup(func() {
		cleanCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		muts := []*spanner.Mutation{
			spanner.Delete("OntologyGraphEdges", spanner.KeyRange{Start: spanner.Key{cellID}, End: spanner.Key{cellID}, Kind: spanner.ClosedClosed}),
			spanner.Delete("OntologyObjects", spanner.KeyRange{Start: spanner.Key{cellID}, End: spanner.Key{cellID}, Kind: spanner.ClosedClosed}),
		}
		for _, id := range outboxIDs {
			muts = append(muts, spanner.Delete("OutboxEvents", spanner.Key{id}))
		}
		_, _ = client.Apply(cleanCtx, muts)
	})
}

func shortTestID(prefix, suffix string) string {
	if len(suffix) > 10 {
		suffix = suffix[len(suffix)-10:]
	}
	return prefix + "-" + suffix
}

func TestLiveSpanner_UpsertMonotonicChunkingAndOutbox(t *testing.T) {
	client := openOntologySpannerClient(t)
	ctx := context.Background()
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	cellID := shortTestID("c-ont", suffix)
	supID := shortTestID("sup-ont", suffix)
	cleanupOntologyCellRows(t, client, cellID, nil)

	repo := NewSpannerRepository(client)
	svc := NewService(repo, nil)
	svc.SetDefaultCell(cellID)

	// 1. Upsert v5 Order + Shop + Edge with EmitOutbox: true
	_, err := svc.UpsertDirect(ctx, []*OntologyObject{
		{
			CellID:           cellID,
			ObjectType:       TypeShop,
			ObjectID:         "shop-" + suffix,
			SupplierID:       supID,
			Version:          1,
			SecurityMarkings: []string{MarkingPII},
			Properties:       map[string]any{"retailer_id": "shop-" + suffix, "name": "Live Shop", "phone": "+998901112233"},
		},
		{
			CellID:           cellID,
			ObjectType:       TypeOrder,
			ObjectID:         "ord-" + suffix,
			SupplierID:       supID,
			Version:          5,
			SecurityMarkings: []string{MarkingFinanceOnly},
			Properties:       map[string]any{"order_id": "ord-" + suffix, "retailer_id": "shop-" + suffix, "status": "DELIVERED", "total_minor": int64(420000), "currency": "UZS"},
		},
	}, []*OntologyGraphEdge{
		{
			CellID:           cellID,
			SourceObjectType: TypeShop,
			SourceObjectID:   "shop-" + suffix,
			LinkType:         LinkOrdered,
			TargetObjectType: TypeOrder,
			TargetObjectID:   "ord-" + suffix,
			SupplierID:       supID,
		},
	}, UpsertOptions{StrictMonotonic: true, EmitOutbox: true, ChangeKind: "LIVE_TEST"})
	require.NoError(t, err)

	// 2. Stale v2 Order update must be skipped cleanly without regressing status
	stats, err := svc.UpsertDirect(ctx, []*OntologyObject{
		{
			CellID:     cellID,
			ObjectType: TypeOrder,
			ObjectID:   "ord-" + suffix,
			SupplierID: supID,
			Version:    2,
			Properties: map[string]any{"order_id": "ord-" + suffix, "retailer_id": "shop-" + suffix, "status": "CREATED", "total_minor": int64(100), "currency": "UZS"},
		},
	}, nil, UpsertOptions{StrictMonotonic: true, EmitOutbox: true})
	require.NoError(t, err)
	require.Equal(t, 0, stats.ObjectsUpserted)
	require.Equal(t, 1, stats.ObjectsSkipped)

	got, _, err := svc.GetObject(ctx, cellID, TypeOrder, "ord-"+suffix, auth.SupplierRoleOwner)
	require.NoError(t, err)
	require.Equal(t, int64(5), got.Version)
	require.Equal(t, "DELIVERED", got.Properties["status"])

	// 3. Verify OutboxEvents emitted EventOntologyObjectChanged -> TopicIntel
	ro := client.ReadOnlyTransaction().WithTimestampBound(spanner.StrongRead())
	defer ro.Close()
	iter := ro.Query(ctx, spanner.Statement{
		SQL:    `SELECT EventId, AggregateType, TopicName, Payload FROM OutboxEvents WHERE AggregateId = @aid`,
		Params: map[string]any{"aid": fmt.Sprintf("%s/%s/%s", cellID, TypeOrder, "ord-"+suffix)},
	})
	defer iter.Stop()
	var outboxIDs []string
	var foundChanged bool
	for {
		row, rerr := iter.Next()
		if rerr == iterator.Done {
			break
		}
		require.NoError(t, rerr)
		var evID, aggType, topicName string
		var payloadBytes []byte
		require.NoError(t, row.Columns(&evID, &aggType, &topicName, &payloadBytes))
		outboxIDs = append(outboxIDs, evID)
		var ev events.OntologyObjectChangedEvent
		if json.Unmarshal(payloadBytes, &ev) == nil &&
			ev.Type == events.EventOntologyObjectChanged &&
			aggType == events.AggregateOntologyObject &&
			topicName == events.TopicIntel {
			foundChanged = true
		}
	}
	require.True(t, foundChanged, "expected EventOntologyObjectChanged -> TopicIntel in OutboxEvents")
	require.Len(t, outboxIDs, 1, "stale v2 upsert must not emit extra outbox event")
	t.Cleanup(func() {
		if len(outboxIDs) > 0 {
			var dm []*spanner.Mutation
			for _, id := range outboxIDs {
				dm = append(dm, spanner.Delete("OutboxEvents", spanner.Key{id}))
			}
			_, _ = client.Apply(context.Background(), dm)
		}
	})

	// 4. Verify <=900-mutation chunking on live Spanner (550 objects * 2 muts = 1100 > 900 -> 2 chunks)
	bulk := make([]*OntologyObject, 0, 550)
	for i := 0; i < 550; i++ {
		wid := fmt.Sprintf("wh-%s-%03d", suffix, i)
		bulk = append(bulk, &OntologyObject{
			CellID:     cellID,
			ObjectType: TypeWarehouse,
			ObjectID:   wid,
			SupplierID: supID,
			Version:    1,
			Properties: map[string]any{"warehouse_id": wid, "name": "Bulk Hub"},
		})
	}
	bulkStats, err := svc.UpsertDirect(ctx, bulk, nil, UpsertOptions{EmitOutbox: false})
	require.NoError(t, err)
	require.Equal(t, 550, bulkStats.ObjectsUpserted)

	// 5. Verify GetNeighbors & viewer redaction on Spanner
	neigh, err := svc.GetNeighbors(ctx, cellID, supID, TypeOrder, "ord-"+suffix, 2, nil, auth.SupplierRoleAgent)
	require.NoError(t, err)
	require.True(t, neigh.PIIMasked)
	require.Equal(t, 1, neigh.TotalNodes)
	require.Equal(t, RedactedPlaceholder, neigh.Nodes[0].Object.Properties["phone"])
}

func TestLiveSpanner_RebuildFromDomainTablesAndUnifiedKG(t *testing.T) {
	client := openOntologySpannerClient(t)
	ctx := context.Background()
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	cellID := shortTestID("c-rb", suffix)
	supID := shortTestID("sup-rb", suffix)
	facID := "fac-rb-" + suffix
	whID := "wh-rb-" + suffix
	prodID := "prod-rb-" + suffix
	lotID := "lot-rb-" + suffix
	retID := "ret-rb-" + suffix
	vehID := "veh-rb-" + suffix
	drvID := "drv-rb-" + suffix
	ordID := "ord-rb-" + suffix
	manID := "man-rb-" + suffix
	invID := "inv-rb-" + suffix
	payID := "pay-rb-" + suffix
	rcptID := "rcpt-rb-" + suffix

	lineItemsJSON, err := json.Marshal([]map[string]any{{
		"product_id":       prodID,
		"sku":              "SKU-RB-" + suffix,
		"name":             "Milk 1L",
		"quantity":         4,
		"unit_price_minor": 15000,
		"line_total_minor": 60000,
	}})
	require.NoError(t, err)

	_, err = client.Apply(ctx, []*spanner.Mutation{
		spanner.InsertMap("Suppliers", map[string]any{
			"SupplierId": supID, "Name": "Samarkand Dairy", "CountryCode": "UZ", "Currency": "UZS",
			"IsConfigured": true, "CreatedAt": spanner.CommitTimestamp, "UpdatedAt": spanner.CommitTimestamp,
		}),
		spanner.InsertMap("SupplierProfiles", map[string]any{
			"SupplierId": supID, "ContactName": "Dairy Ops", "TaxId": "301223344",
			"Email": "ops@dairy.uz", "Phone": "+998901112233", "RegisteredAt": spanner.CommitTimestamp, "UpdatedAt": spanner.CommitTimestamp,
		}),
		spanner.InsertMap("Factories", map[string]any{
			"FactoryId": facID, "SupplierId": supID, "Name": "Dairy Plant 1",
			"CountryCode": "UZ", "IsActive": true, "CreatedAt": spanner.CommitTimestamp, "UpdatedAt": spanner.CommitTimestamp,
		}),
		spanner.InsertMap("Warehouses", map[string]any{
			"WarehouseId": whID, "SupplierId": supID, "Name": "Cold Depot", "PrimaryFactoryId": facID,
			"CountryCode": "UZ", "CoverageRadiusKm": float64(25.0), "IsActive": true, "IsOnShift": true, "AutoDispatchEnabled": true,
			"CreatedAt": spanner.CommitTimestamp, "UpdatedAt": spanner.CommitTimestamp,
		}),
		spanner.InsertMap("Products", map[string]any{
			"ProductId": prodID, "SupplierId": supID, "CategoryId": "cat-dairy", "Name": "Milk 1L",
			"PriceMinor": int64(15000), "Currency": "UZS", "StockQuantity": int64(100), "IsActive": true,
			"CreatedAt": spanner.CommitTimestamp, "UpdatedAt": spanner.CommitTimestamp,
		}),
		spanner.InsertMap("StockLots", map[string]any{
			"LotId": lotID, "SupplierId": supID, "WarehouseId": whID, "ProductId": prodID,
			"LocationId": "bin-1", "LotCode": "LOT-001", "QuantityOnHand": int64(80), "QuantityReserved": int64(10),
			"Status": "AVAILABLE", "ReceivedAt": spanner.CommitTimestamp,
			"CreatedAt": spanner.CommitTimestamp, "UpdatedAt": spanner.CommitTimestamp,
		}),
		spanner.InsertMap("Retailers", map[string]any{
			"RetailerId": retID, "Phone": "+998905556677", "Name": "Baraka Store",
			"CountryCode": "UZ", "CreatedAt": spanner.CommitTimestamp,
		}),
		spanner.InsertMap("Vehicles", map[string]any{
			"VehicleId": vehID, "Label": "Isuzu-01", "LicensePlate": "01A777BB", "SupplierId": supID,
			"HomeNodeType": "WAREHOUSE", "HomeNodeId": whID, "VehicleClass": "VAN",
			"MaxVolumeVu": float64(500), "IsActive": true, "CreatedAt": spanner.CommitTimestamp, "UpdatedAt": spanner.CommitTimestamp,
		}),
		spanner.InsertMap("Drivers", map[string]any{
			"DriverId": drvID, "Name": "Rustam Driver", "Phone": "+998908889900", "SupplierId": supID,
			"HomeNodeType": "WAREHOUSE", "HomeNodeId": whID, "VehicleId": vehID,
			"IsActive": true, "OnShift": true, "CreatedAt": spanner.CommitTimestamp, "UpdatedAt": spanner.CommitTimestamp,
		}),
		spanner.InsertMap("Orders", map[string]any{
			"OrderId": ordID, "SupplierId": supID, "RetailerId": retID, "WarehouseId": whID,
			"DriverId": drvID, "VehicleId": vehID, "Status": "IN_TRANSIT", "OrderSource": "MOBILE", "ConfirmationStatus": "AUTO",
			"TotalMinor": int64(60000), "Currency": "UZS", "LineItemsJson": lineItemsJSON, "Version": int64(1),
			"CreatedAt": spanner.CommitTimestamp, "UpdatedAt": spanner.CommitTimestamp,
		}),
		spanner.InsertMap("SupplierTruckManifests", map[string]any{
			"ManifestId": manID, "SupplierId": supID, "WarehouseId": whID, "TruckId": vehID, "DriverId": drvID,
			"State": "DISPATCHED", "CreatedAt": spanner.CommitTimestamp, "UpdatedAt": spanner.CommitTimestamp,
		}),
		spanner.InsertMap("ManifestOrders", map[string]any{
			"ManifestId": manID, "OrderId": ordID, "State": "LOADED", "SequenceIndex": int64(1), "UpdatedAt": spanner.CommitTimestamp,
		}),
		spanner.InsertMap("ArInvoices", map[string]any{
			"InvoiceId": invID, "SupplierId": supID, "RetailerId": retID, "OrderId": ordID,
			"PrincipalMinor": int64(60000), "BalanceMinor": int64(60000), "Currency": "UZS",
			"TermsDays": int64(14), "GracePeriodDays": int64(3), "Status": "OPEN",
			"AgingBucket": "CURRENT", "DunningStep": int64(0),
			"CreditLeaveAt": time.Now().UTC(), "DueAt": time.Now().UTC().Add(14 * 24 * time.Hour),
			"CreatedAt": spanner.CommitTimestamp, "UpdatedAt": spanner.CommitTimestamp,
		}),
		spanner.InsertMap("RetailerCreditProfiles", map[string]any{
			"RetailerId": retID, "SupplierId": supID,
			"CreditLimitMinor": int64(500000), "CurrentBalanceMinor": int64(60000), "ReservedMinor": int64(0),
			"AvailableCreditMinor": int64(440000), "Status": "ACTIVE", "Version": int64(1),
			"CreatedAt": spanner.CommitTimestamp, "UpdatedAt": spanner.CommitTimestamp,
		}),
		spanner.InsertMap("PaymentSessions", map[string]any{
			"SessionId": payID, "OrderId": ordID, "SupplierId": supID, "RetailerId": retID,
			"Gateway": "payme", "Mode": "ONLINE", "Status": "CLEARED",
			"AmountMinor": int64(60000), "Currency": "UZS", "CreatedAt": spanner.CommitTimestamp, "UpdatedAt": spanner.CommitTimestamp,
		}),
		spanner.InsertMap("OrderFiscalReceipts", map[string]any{
			"AttemptId": rcptID, "OrderId": ordID, "SupplierId": supID, "Provider": "SOLIQ_OFD",
			"Status": "SUCCEEDED", "AmountMinor": int64(60000), "Currency": "UZS",
			"CreatedAt": spanner.CommitTimestamp, "UpdatedAt": spanner.CommitTimestamp,
		}),
	})
	require.NoError(t, err)

	t.Cleanup(func() {
		cleanCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		muts := []*spanner.Mutation{
			spanner.Delete("OntologyGraphEdges", spanner.KeyRange{Start: spanner.Key{cellID}, End: spanner.Key{cellID}, Kind: spanner.ClosedClosed}),
			spanner.Delete("OntologyObjects", spanner.KeyRange{Start: spanner.Key{cellID}, End: spanner.Key{cellID}, Kind: spanner.ClosedClosed}),
			spanner.Delete("OrderFiscalReceipts", spanner.Key{ordID, rcptID}),
			spanner.Delete("PaymentSessions", spanner.Key{payID}),
			spanner.Delete("RetailerCreditProfiles", spanner.Key{retID, supID}),
			spanner.Delete("ArInvoices", spanner.Key{invID}),
			spanner.Delete("ManifestOrders", spanner.Key{manID, ordID}),
			spanner.Delete("SupplierTruckManifests", spanner.Key{manID}),
			spanner.Delete("Orders", spanner.Key{ordID}),
			spanner.Delete("Drivers", spanner.Key{drvID}),
			spanner.Delete("Vehicles", spanner.Key{vehID}),
			spanner.Delete("Retailers", spanner.Key{retID}),
			spanner.Delete("StockLots", spanner.Key{lotID}),
			spanner.Delete("Products", spanner.Key{prodID}),
			spanner.Delete("Warehouses", spanner.Key{whID}),
			spanner.Delete("Factories", spanner.Key{facID}),
			spanner.Delete("SupplierProfiles", spanner.Key{supID}),
			spanner.Delete("Suppliers", spanner.Key{supID}),
		}
		_, _ = client.Apply(cleanCtx, muts)
	})

	svc := NewService(NewSpannerRepository(client), nil)
	svc.SetDefaultCell(cellID)

	rbRes, err := svc.RebuildFromSpanner(ctx, cellID, supID)
	require.NoError(t, err)
	require.Equal(t, "COMPLETED", rbRes.Status)
	require.GreaterOrEqual(t, rbRes.ObjectsUpserted, 14)
	require.GreaterOrEqual(t, rbRes.EdgesUpserted, 10)
	require.Contains(t, rbRes.TablesScanned, "StockLots")
	require.Contains(t, rbRes.TablesScanned, "ArInvoices")
	require.Contains(t, rbRes.TablesScanned, "OrderFiscalReceipts")

	// Clean up OutboxEvents created by rebuild
	ro := client.ReadOnlyTransaction().WithTimestampBound(spanner.StrongRead())
	obIter := ro.Query(ctx, spanner.Statement{
		SQL:    `SELECT EventId FROM OutboxEvents WHERE AggregateType = @agg AND STARTS_WITH(AggregateId, @prefix)`,
		Params: map[string]any{"agg": events.AggregateOntologyObject, "prefix": cellID + "/"},
	})
	var obIDs []string
	for {
		row, rerr := obIter.Next()
		if rerr == iterator.Done {
			break
		}
		if rerr == nil {
			var id string
			if row.Column(0, &id) == nil {
				obIDs = append(obIDs, id)
			}
		}
	}
	obIter.Stop()
	ro.Close()
	t.Cleanup(func() {
		var dm []*spanner.Mutation
		for _, id := range obIDs {
			dm = append(dm, spanner.Delete("OutboxEvents", spanner.Key{id}))
		}
		if len(dm) > 0 {
			_, _ = client.Apply(context.Background(), dm)
		}
	})

	// Second rebuild must be idempotent (0 objects upserted when zero drift)
	rbAgain, err := svc.RebuildFromSpanner(ctx, cellID, supID)
	require.NoError(t, err)
	require.Equal(t, 0, rbAgain.ObjectsUpserted)

	// Unified Knowledge Graph honors security markings
	kgOwner, err := svc.GetKnowledgeGraphUnified(ctx, cellID, supID, auth.SupplierRoleOwner)
	require.NoError(t, err)
	require.Equal(t, "unified_ontology", kgOwner.Source)
	require.GreaterOrEqual(t, kgOwner.OntologyObjectsCount, 14)

	kgAgent, err := svc.GetKnowledgeGraphUnified(ctx, cellID, supID, auth.SupplierRoleAgent)
	require.NoError(t, err)
	for _, n := range kgAgent.Nodes {
		if n.ID == drvID || n.Type == "legal_entity" {
			require.Equal(t, RedactedPlaceholder, n.Name)
		}
	}
}

func TestLiveKafkaToSpanner_OntologyProjectorEndToEnd(t *testing.T) {
	client := openOntologySpannerClient(t)
	requireKafka := strings.TrimSpace(os.Getenv("PARITY_REQUIRE_KAFKA")) == "1"
	rawBrokers := strings.TrimSpace(os.Getenv("KAFKA_BROKERS"))
	if rawBrokers == "" {
		rawBrokers = "localhost:9092"
	}
	brokers := strings.Split(rawBrokers, ",")

	dialCtx, dialCancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer dialCancel()
	dialer := &kgo.Dialer{Timeout: 3 * time.Second}
	conn, err := dialer.DialContext(dialCtx, "tcp", brokers[0])
	if err != nil {
		if requireKafka {
			t.Fatalf("PARITY_REQUIRE_KAFKA=1 but kafka %s unreachable: %v", brokers[0], err)
		}
		t.Skipf("kafka %s unavailable (%v); skipping live Kafka ontology projector test", brokers[0], err)
	}
	controller, err := conn.Controller()
	_ = conn.Close()
	if err != nil {
		if requireKafka {
			t.Fatalf("kafka controller failed: %v", err)
		}
		t.Skipf("kafka controller unavailable: %v", err)
	}
	ctrlConn, err := kgo.Dial("tcp", net.JoinHostPort(controller.Host, fmt.Sprintf("%d", controller.Port)))
	if err != nil {
		if requireKafka {
			t.Fatalf("kafka controller dial failed: %v", err)
		}
		t.Skipf("kafka controller dial failed: %v", err)
	}

	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	topic := "ont-live-" + suffix
	groupID := "void-ontology-projector-test-" + suffix
	cellID := shortTestID("c-kfk", suffix)
	supID := shortTestID("sup-kfk", suffix)

	require.NoError(t, ctrlConn.CreateTopics(kgo.TopicConfig{
		Topic:             topic,
		NumPartitions:     1,
		ReplicationFactor: 1,
	}))
	_ = ctrlConn.Close()
	t.Cleanup(func() {
		if c, cerr := kgo.Dial("tcp", net.JoinHostPort(controller.Host, fmt.Sprintf("%d", controller.Port))); cerr == nil {
			_ = c.DeleteTopics(topic)
			_ = c.Close()
		}
	})

	cleanupOntologyCellRows(t, client, cellID, nil)

	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
	defer cancel()

	// Wait until Kafka topic leader is ready
	deadline := time.Now().Add(10 * time.Second)
	for {
		lc, lerr := kgo.DialLeader(ctx, "tcp", brokers[0], topic, 0)
		if lerr == nil {
			_ = lc.Close()
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("topic %s leader not ready: %v", topic, lerr)
		}
		time.Sleep(150 * time.Millisecond)
	}

	svc := NewService(NewSpannerRepository(client), nil)
	svc.SetDefaultCell(cellID)
	dedup := pegasuskafka.NewInMemoryEventDedup(time.Hour)
	handler := pegasuskafka.WithEventDedup(dedup, groupID, NewEventConsumer(svc, nil).HandleEvent)
	consumer := pegasuskafka.NewConsumer(pegasuskafka.ConsumerDeps{
		Brokers: brokers,
		GroupID: groupID,
		Topic:   topic,
		Handler: handler,
	})
	consCtx, consCancel := context.WithCancel(ctx)
	defer consCancel()
	go consumer.Start(consCtx)
	t.Cleanup(func() {
		consCancel()
		_ = consumer.Close()
	})

	writer := &kgo.Writer{
		Addr:     kgo.TCP(brokers...),
		Topic:    topic,
		Balancer: &kgo.LeastBytes{},
	}
	t.Cleanup(func() { _ = writer.Close() })

	ordID := "ord-k-" + suffix
	lotID := "lot-k-" + suffix
	payID := "pay-k-" + suffix
	msgs := []kgo.Message{
		{
			Key:   []byte(ordID),
			Value: []byte(fmt.Sprintf(`{"type":"ORDER_CREATED","version":5,"order_id":%q,"supplier_id":%q,"retailer_id":"shop-k1","status":"CONFIRMED","total_minor":320000,"currency":"UZS","trace_id":"t1-%s"}`, ordID, supID, suffix)),
		},
		{
			Key:   []byte(ordID),
			Value: []byte(fmt.Sprintf(`{"type":"ORDER_STATUS_CHANGED","version":2,"order_id":%q,"supplier_id":%q,"retailer_id":"shop-k1","status":"CANCELLED","total_minor":0,"currency":"UZS","trace_id":"t2-%s"}`, ordID, supID, suffix)),
		},
		{
			Key:   []byte(lotID),
			Value: []byte(fmt.Sprintf(`{"type":"LOT_RECALL_INITIATED","version":1,"lot_id":%q,"supplier_id":%q,"warehouse_id":"wh-k1","product_id":"sku-k1","order_id":%q,"recall_reason":"CONTAMINATION","trace_id":"t3-%s"}`, lotID, supID, ordID, suffix)),
		},
		{
			Key:   []byte(payID),
			Value: []byte(fmt.Sprintf(`{"type":"PAYMENT_CLEARED","version":1,"payment_id":%q,"order_id":%q,"supplier_id":%q,"amount_minor":320000,"gateway":"payme","trace_id":"t4-%s"}`, payID, ordID, supID, suffix)),
		},
	}
	require.NoError(t, writer.WriteMessages(ctx, msgs...))

	// Poll Spanner OntologyObjects until all 3 objects are projected and stale v2 did not overwrite v5
	pollDeadline := time.Now().Add(20 * time.Second)
	for {
		ordObj, _, ordErr := svc.GetObject(ctx, cellID, TypeOrder, ordID, auth.SupplierRoleOwner)
		lotObj, _, lotErr := svc.GetObject(ctx, cellID, TypeLot, lotID, auth.SupplierRoleOwner)
		payObj, _, payErr := svc.GetObject(ctx, cellID, TypePayment, payID, auth.SupplierRoleOwner)
		if ordErr == nil && lotErr == nil && payErr == nil {
			require.Equal(t, int64(5), ordObj.Version)
			require.Equal(t, "CONFIRMED", ordObj.Properties["status"])
			require.Contains(t, lotObj.SecurityMarkings, MarkingLotRecallQuarantine)
			require.Contains(t, payObj.SecurityMarkings, MarkingFinanceOnly)
			break
		}
		if time.Now().After(pollDeadline) {
			t.Fatalf("timed out waiting for Kafka -> Spanner ontology projection (ord=%v, lot=%v, pay=%v)", ordErr, lotErr, payErr)
		}
		time.Sleep(200 * time.Millisecond)
	}

	// Clean up OutboxEvents created by Kafka projection
	ro := client.ReadOnlyTransaction().WithTimestampBound(spanner.StrongRead())
	obIter := ro.Query(ctx, spanner.Statement{
		SQL:    `SELECT EventId FROM OutboxEvents WHERE AggregateType = @agg AND STARTS_WITH(AggregateId, @prefix)`,
		Params: map[string]any{"agg": events.AggregateOntologyObject, "prefix": cellID + "/"},
	})
	var obIDs []string
	for {
		row, rerr := obIter.Next()
		if rerr == iterator.Done {
			break
		}
		if rerr == nil {
			var id string
			if row.Column(0, &id) == nil {
				obIDs = append(obIDs, id)
			}
		}
	}
	obIter.Stop()
	ro.Close()
	require.GreaterOrEqual(t, len(obIDs), 3)
	t.Cleanup(func() {
		var dm []*spanner.Mutation
		for _, id := range obIDs {
			dm = append(dm, spanner.Delete("OutboxEvents", spanner.Key{id}))
		}
		if len(dm) > 0 {
			_, _ = client.Apply(context.Background(), dm)
		}
	})
}

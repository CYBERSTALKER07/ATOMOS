package bootstrap

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/audit"
	"github.com/pegasusx/pegasusx/apps/backend-go/cache"
	"github.com/pegasusx/pegasusx/apps/backend-go/connectors"
	"github.com/pegasusx/pegasusx/apps/backend-go/kafkautil"
	"github.com/pegasusx/pegasusx/apps/backend-go/ontology"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
)

func TestSetupWave3C_MemoryFallbackAndOntologyHook(t *testing.T) {
	ctx := context.Background()
	auditSvc := audit.NewServiceWithRepository(audit.NewMemoryRepository())
	ontSvc := ontology.NewService(ontology.NewMemoryRepository(), nil)
	app := &App{
		Audit:    auditSvc,
		Ontology: ontSvc,
	}
	cfg := &Config{
		AllowMemoryFallback: true,
		JWTSecret:           "wave3c-wiring-memory-secret-123456",
		JWTIssuer:           "pegasusx-wave3c-test",
	}
	if err := SetupWave3C(ctx, cfg, nil, app, nil); err != nil {
		t.Fatalf("SetupWave3C(memory): %v", err)
	}
	if app.Connectors == nil || app.ExportPacks == nil {
		t.Fatalf("expected Connectors and ExportPacks initialized in memory fallback")
	}

	// Seed an Ontology SKU object and verify BuildConnectorOntologyLineageHook enriches source_system.
	_, err := ontSvc.UpsertDirect(ctx, []*ontology.OntologyObject{{
		CellID:     ontology.DefaultCellID,
		ObjectType: ontology.TypeSKU,
		ObjectID:   "SKU-W3C-1",
		SupplierID: "sup-w3c-mem",
		Version:    1,
		Properties: map[string]any{
			"product_id":  "SKU-W3C-1",
			"sku":         "SKU-W3C-1",
			"name":        "Green Tea",
			"price_minor": int64(1200000),
			"currency":    "UZS",
		},
		SecurityMarkings: []string{ontology.MarkingFinanceOnly},
	}}, nil, ontology.UpsertOptions{StrictMonotonic: true})
	if err != nil {
		t.Fatalf("seed ontology SKU: %v", err)
	}

	hook := BuildConnectorOntologyLineageHook(ontSvc)
	if hook == nil {
		t.Fatalf("expected non-nil ontology lineage hook")
	}
	if err := hook(ctx, connectors.OntologyLineageInput{
		TenantID: "sup-w3c-mem",
		Provider: "billz",
		RunID:    "crun_mem_1",
		SyncedAt: time.Now().UTC(),
		EntityMaps: []connectors.ConnectorEntityMap{
			{
				TenantID:   "sup-w3c-mem",
				Provider:   "billz",
				ObjectType: connectors.ObjectTypeProduct,
				ExternalID: "BILLZ-9001",
				InternalID: "SKU-W3C-1",
			},
			{
				TenantID:   "sup-w3c-mem",
				Provider:   "billz",
				ObjectType: connectors.ObjectTypeProduct,
				ExternalID: "BILLZ-MISSING",
				InternalID: "SKU-NOT-IN-ONTOLOGY",
			},
		},
	}); err != nil {
		t.Fatalf("lineage hook error: %v", err)
	}

	got, _, err := ontSvc.GetObject(ctx, ontology.DefaultCellID, ontology.TypeSKU, "SKU-W3C-1", "ADMIN")
	if err != nil {
		t.Fatalf("GetObject after lineage hook: %v", err)
	}
	if got.Properties["source_system"] != "billz" || got.Properties["external_id"] != "BILLZ-9001" || got.Properties["connector_run_id"] != "crun_mem_1" {
		t.Fatalf("unexpected ontology lineage properties: %+v", got.Properties)
	}
	if got.Version < 2 {
		t.Fatalf("expected monotonic version >= 2, got %d", got.Version)
	}
}

func TestNewApp_WiresAllWave3CModulesAndHooks_LiveSpanner(t *testing.T) {
	host := strings.TrimSpace(os.Getenv("SPANNER_EMULATOR_HOST"))
	if host == "" {
		if strings.TrimSpace(os.Getenv("PARITY_REQUIRE_SPANNER")) == "1" {
			t.Fatal("PARITY_REQUIRE_SPANNER=1 but SPANNER_EMULATOR_HOST is unset")
		}
		t.Skip("SPANNER_EMULATOR_HOST not set; skipping bootstrap live Spanner Wave 3C wiring test")
	}

	stubRuntimeConstructors(t)
	newRedisRuntimeAdapter = func(_ cache.RedisConfig) (redisRuntimeAdapter, error) {
		return &fakeRedisAdapter{}, nil
	}
	newKafkaRuntimePublisher = func(_ string, _ outbox.KafkaPublisherConfig) (kafkaRuntimePublisher, error) {
		return &fakeKafkaPublisher{}, nil
	}
	newKafkaRuntimeDLQWriter = func(_ string, _ string, _ kafkautil.ClientAuth) (kafkaRuntimeDLQWriter, error) {
		return &fakeKafkaDLQWriter{}, nil
	}
	newSpannerRuntimeClient = func(ctx context.Context, dsn string) (*spanner.Client, error) {
		c, err := spanner.NewClient(ctx, dsn,
			option.WithEndpoint(host),
			option.WithoutAuthentication(),
			option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())),
		)
		if err != nil {
			return nil, err
		}
		ro := c.ReadOnlyTransaction()
		_, eErr := ro.ReadRow(ctx, "ConnectorSyncRuns", spanner.Key{"__bs_w3c__", "__probe__"}, []string{"Status"})
		ro.Close()
		if eErr != nil && spanner.ErrCode(eErr) != codes.NotFound {
			c.Close()
			return nil, eErr
		}
		return c, nil
	}

	cfg := testConfig()
	cfg.RunMode = RunModeAPI
	cfg.RequireInfraAdapters = false
	cfg.JWTSecret = "wave3c-live-spanner-bootstrap-secret-123456"
	cfg.JWTIssuer = "pegasusx-bootstrap-test"

	app, err := NewApp(context.Background(), cfg)
	if err != nil {
		t.Fatalf("NewApp(api + live Spanner Wave 3C): %v", err)
	}
	defer app.Close()

	if app.Connectors == nil || app.Connectors.Repo() == nil {
		t.Fatalf("expected app.Connectors wired with live Spanner repository")
	}
	if app.ExportPacks == nil {
		t.Fatalf("expected app.ExportPacks wired")
	}

	ctx := context.Background()
	tenantID := "sup-bs-w3c-" + strings.ReplaceAll(uuid.NewString(), "-", "")[:10]
	t.Setenv("CONNECTOR_SECRET_BS_W3C_BILLZ_TOKEN", "bs-w3c-secret-token")

	t.Cleanup(func() {
		if app.Spanner == nil {
			return
		}
		_, _ = app.Spanner.Apply(context.Background(), []*spanner.Mutation{
			spanner.Delete("ConnectorSyncRuns", spanner.KeyRange{Start: spanner.Key{tenantID}, End: spanner.Key{tenantID}, Kind: spanner.ClosedClosed}),
			spanner.Delete("ExternalCredentials", spanner.KeyRange{Start: spanner.Key{tenantID}, End: spanner.Key{tenantID}, Kind: spanner.ClosedClosed}),
			spanner.Delete("OutboxEvents", spanner.KeyRange{Start: spanner.Key{}, End: spanner.Key{}}),
		})
	})

	cred, err := app.Connectors.UpsertCredential(ctx, connectors.UpsertCredentialInput{
		TenantID:     tenantID,
		Provider:     "billz",
		AccountLabel: "primary",
		AuthType:     "bearer",
		SecretRef:    "env://CONNECTOR_SECRET_BS_W3C_BILLZ_TOKEN",
		BaseURL:      "https://billz.example.test",
		CreatedBy:    "bootstrap-owner",
		ActorType:    "human",
		ActorRole:    "OWNER",
	})
	if err != nil {
		t.Fatalf("app.Connectors.UpsertCredential: %v", err)
	}

	// Verify AuditRecorder hook recorded an AuditEvents row in live Spanner.
	evs, err := app.Audit.ListEvents(ctx, audit.Filter{
		TenantID:   tenantID,
		ObjectType: "ExternalCredential",
		ObjectID:   cred.CredentialID,
		ViewerRole: "ADMIN",
	})
	if err != nil {
		t.Fatalf("app.Audit.ListEvents: %v", err)
	}
	if len(evs) == 0 {
		t.Fatalf("expected AuditEvents row persisted via NewConnectorsAuditRecorder for %s", cred.CredentialID)
	}
	raw, _ := json.Marshal(cred)
	if strings.Contains(string(raw), "bs-w3c-secret-token") {
		t.Fatalf("UpsertCredential summary leaked raw token: %s", string(raw))
	}
}

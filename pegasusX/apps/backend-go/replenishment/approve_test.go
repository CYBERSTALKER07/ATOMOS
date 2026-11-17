package replenishment

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"cloud.google.com/go/civil"
	"cloud.google.com/go/spanner"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
)

func newReplEmulatorClient(t *testing.T, ctx context.Context) *spanner.Client {
	t.Helper()
	host := strings.TrimSpace(os.Getenv("SPANNER_EMULATOR_HOST"))
	if host == "" {
		if os.Getenv("PARITY_REQUIRE_SPANNER") == "1" {
			t.Fatal("PARITY_REQUIRE_SPANNER=1 but SPANNER_EMULATOR_HOST is unset")
		}
		t.Skip("SPANNER_EMULATOR_HOST not set; skipping live Spanner test")
	}
	envOr := func(k, def string) string {
		if v := os.Getenv(k); v != "" {
			return v
		}
		return def
	}
	dbPath := fmt.Sprintf("projects/%s/instances/%s/databases/%s",
		envOr("SPANNER_PROJECT", "pegasusx-local"),
		envOr("SPANNER_INSTANCE", "pegasusx-instance"),
		envOr("SPANNER_DATABASE", "pegasusx-db"))
	cli, err := spanner.NewClient(ctx, dbPath,
		option.WithEndpoint(host), option.WithoutAuthentication(),
		option.WithGRPCDialOption(grpc.WithInsecure()))
	if err != nil {
		t.Fatalf("spanner client: %v", err)
	}
	return cli
}

func TestHandleApprove_LiveSpannerSuggestionApproveAndOutbox(t *testing.T) {
	ctx := context.Background()
	client := newReplEmulatorClient(t, ctx)
	defer client.Close()

	tag := strings.ReplaceAll(uuid.NewString(), "-", "")[:8]
	retID := "rr-" + tag
	supID := "rs-" + tag
	skuID := "sku-" + tag

	defer func() {
		_, _ = client.Apply(context.Background(), []*spanner.Mutation{
			spanner.Delete("ReorderSuggestions", spanner.Key{retID, skuID}),
			spanner.Delete("RetailerPaymentTerms", spanner.Key{retID, supID}),
		})
	}()

	_, err := client.Apply(ctx, []*spanner.Mutation{
		spanner.InsertOrUpdateMap("RetailerPaymentTerms", map[string]any{
			"RetailerId": retID, "SupplierId": supID, "CreditEnabled": true, "TermsDays": int64(30),
			"GracePeriodDays": int64(0), "CreditLimitMinor": int64(500_000), "UseGlobalDefaults": false,
			"ConsentVerified": true, "Version": int64(1), "UpdatedAt": spanner.CommitTimestamp,
		}),
		spanner.InsertOrUpdateMap("ReorderSuggestions", map[string]any{
			"RetailerId":      retID,
			"Sku":             skuID,
			"SuggestedQty":    int64(24),
			"AdjustedDemand":  12.0,
			"CurrentStock":    int64(4),
			"InFlightQty":     int64(0),
			"SafetyStock":     8.0,
			"SuggestedByDate": civil.DateOf(time.Now().UTC().AddDate(0, 0, 2)),
			"ComputedAt":      time.Now().UTC(),
			"Status":          "OPEN",
		}),
	})
	if err != nil {
		t.Fatalf("seed ReorderSuggestions/Link: %v", err)
	}

	api := NewSuggestionsAPI(client, nil, nil)
	r := chi.NewRouter()
	r.Post("/v1/replenishment/suggestions/{id}/approve", api.HandleApprove)

	// 1. Agent actor forbidden.
	agentCtx := auth.WithClaims(ctx, auth.Claims{
		Subject: "agent:bot-1", Role: auth.RoleAdmin, SupplierID: supID, ActorType: "agent",
	})
	reqAgent := httptest.NewRequest(http.MethodPost, "/v1/replenishment/suggestions/"+retID+":"+skuID+"/approve", strings.NewReader(`{"mode":"approve_only"}`)).WithContext(agentCtx)
	rrAgent := httptest.NewRecorder()
	r.ServeHTTP(rrAgent, reqAgent)
	if rrAgent.Code != http.StatusForbidden {
		t.Fatalf("expected 403 agent approve, got %d body=%s", rrAgent.Code, rrAgent.Body.String())
	}

	// 2. Human supplier admin approves with overridden qty=30 (mode=approve_only since OrderSvc=nil).
	supCtx := auth.WithClaims(ctx, auth.Claims{
		Subject: "usr-sup-1", Role: auth.RoleAdmin, SupplierID: supID, ActorType: "human",
	})
	reqOK := httptest.NewRequest(http.MethodPost, "/v1/replenishment/suggestions/"+retID+":"+skuID+"/approve", strings.NewReader(`{"qty":30,"mode":"approve_only"}`)).WithContext(supCtx)
	rrOK := httptest.NewRecorder()
	r.ServeHTTP(rrOK, reqOK)
	if rrOK.Code != http.StatusOK {
		t.Fatalf("expected 200 approve, got %d body=%s", rrOK.Code, rrOK.Body.String())
	}
	var resp ApproveSuggestionResponse
	if err := json.Unmarshal(rrOK.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.Status != SuggestionStatusApproved || resp.ApprovedQty != 30 {
		t.Fatalf("unexpected approve response: %+v", resp)
	}

	// 3. Re-approving same suggestion returns idempotent AlreadyApproved=true.
	reqIdem := httptest.NewRequest(http.MethodPost, "/v1/replenishment/suggestions/"+retID+":"+skuID+"/approve", strings.NewReader(`{}`)).WithContext(supCtx)
	rrIdem := httptest.NewRecorder()
	r.ServeHTTP(rrIdem, reqIdem)
	if rrIdem.Code != http.StatusOK {
		t.Fatalf("expected 200 idempotent approve, got %d", rrIdem.Code)
	}
	var idemResp ApproveSuggestionResponse
	_ = json.Unmarshal(rrIdem.Body.Bytes(), &idemResp)
	if !idemResp.AlreadyApproved {
		t.Fatalf("expected AlreadyApproved=true on repeat call, got %+v", idemResp)
	}

	// 4. Verify live Spanner OutboxEvents emitted replenishment.suggestion.approved.
	ro := client.ReadOnlyTransaction()
	defer ro.Close()
	obIter := ro.Query(ctx, spanner.Statement{
		SQL:    `SELECT CAST(Payload AS STRING) FROM OutboxEvents WHERE AggregateId = @aid`,
		Params: map[string]any{"aid": retID + ":" + skuID},
	})
	found := false
	for {
		row, err := obIter.Next()
		if err == iterator.Done || err != nil {
			break
		}
		var payload string
		if row.Columns(&payload) == nil && strings.Contains(payload, "replenishment.suggestion.approved") {
			found = true
			break
		}
	}
	obIter.Stop()
	if !found {
		t.Fatalf("expected replenishment.suggestion.approved outbox event for %s:%s", retID, skuID)
	}
}

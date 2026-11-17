package e2ep0

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/ar"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/gl"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
)

func TestE2E_GLRoutesMountedAndRBACEnforced(t *testing.T) {
	h := NewTestHarness(t)
	if h.GLSvc == nil {
		t.Fatal("expected TestHarness.GLSvc to be wired")
	}

	// Unauthenticated -> 401
	unauth := h.Client.Get("/v1/finance/gl/trial-balance", "")
	if unauth.StatusCode != http.StatusUnauthorized {
		t.Fatalf("unauthenticated trial-balance status = %d, want 401", unauth.StatusCode)
	}

	// Non-admin role (RETAILER) -> 403
	retTok := h.IssueRetailerToken(DefaultSupplierA, "ret-user-1")
	forbidden := h.Client.Get("/v1/finance/gl/trial-balance", retTok)
	if forbidden.StatusCode != http.StatusForbidden {
		t.Fatalf("retailer trial-balance status = %d, want 403", forbidden.StatusCode)
	}

	// Admin -> manual balanced posting -> trial-balance -> reverse -> duplicate reverse 409
	admTok := h.IssueAdminToken(DefaultSupplierA, "adm-user-1")
	postResp := h.Client.Post("/v1/finance/journal/manual", map[string]any{
		"source_id":       "e2e-man-1",
		"currency":        "UZS",
		"idempotency_key": "e2e-man-idem-1",
		"memo":            "E2E smoke entry",
		"postings": []map[string]any{
			{"account_code": "CASH:OFFICE", "direction": "DEBIT", "amount_minor": 50000},
			{"account_code": "REV:SALES", "direction": "CREDIT", "amount_minor": 50000},
		},
	}, admTok)
	if postResp.StatusCode != http.StatusCreated {
		t.Fatalf("manual post status = %d body=%s", postResp.StatusCode, string(postResp.Body))
	}
	var posted gl.JournalEntryResult
	if err := json.Unmarshal(postResp.Body, &posted); err != nil {
		t.Fatalf("decode posted result: %v", err)
	}

	tbResp := h.Client.Get("/v1/finance/gl/trial-balance", admTok)
	if tbResp.StatusCode != http.StatusOK {
		t.Fatalf("admin trial-balance status = %d body=%s", tbResp.StatusCode, string(tbResp.Body))
	}
	var tb gl.TrialBalanceReport
	if err := json.Unmarshal(tbResp.Body, &tb); err != nil {
		t.Fatalf("decode trial balance: %v", err)
	}
	if !tb.Balanced || tb.TotalDebitsMinor != 50000 {
		t.Fatalf("unexpected TB: %+v", tb)
	}

	revResp := h.Client.Post("/v1/finance/journal/"+posted.EntryID+"/reverse", map[string]any{
		"reason":          "e2e reversal check",
		"idempotency_key": "e2e-rev-1",
	}, admTok)
	if revResp.StatusCode != http.StatusCreated {
		t.Fatalf("reverse status = %d body=%s", revResp.StatusCode, string(revResp.Body))
	}
	revDup := h.Client.Post("/v1/finance/journal/"+posted.EntryID+"/reverse", map[string]any{
		"reason":          "e2e double reversal",
		"idempotency_key": "e2e-rev-2",
	}, admTok)
	if revDup.StatusCode != http.StatusConflict {
		t.Fatalf("second reverse status = %d, want 409", revDup.StatusCode)
	}
}

func TestE2E_LiveSpanner_GLDualWriteTrialBalanceAndLedgerParity(t *testing.T) {
	requireSpanner := strings.TrimSpace(os.Getenv("PARITY_REQUIRE_SPANNER")) == "1"
	host := strings.TrimSpace(os.Getenv("SPANNER_EMULATOR_HOST"))
	if host == "" {
		if requireSpanner {
			t.Fatal("PARITY_REQUIRE_SPANNER=1 but SPANNER_EMULATOR_HOST is unset")
		}
		t.Skip("SPANNER_EMULATOR_HOST not set; skipping live Spanner e2ep0 GL parity test")
	}

	dbPath := "projects/pegasusx-local/instances/pegasusx-instance/databases/pegasusx-db"
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	client, err := spanner.NewClient(ctx, dbPath,
		option.WithEndpoint(host),
		option.WithoutAuthentication(),
		option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())),
	)
	if err != nil {
		t.Fatalf("open spanner client: %v", err)
	}
	defer client.Close()

	if _, probeErr := client.Single().ReadRow(ctx, "JournalEntries", spanner.Key{"__e2e_gl_probe__", "__e2e_gl_probe__"}, []string{"EntryId"}); probeErr != nil && spanner.ErrCode(probeErr) != codes.NotFound {
		t.Fatalf("probe JournalEntries: %v", probeErr)
	}

	h := NewTestHarnessWithOptions(t, HarnessOptions{SpannerClient: client})
	t.Setenv("AR_INVOICES_ENABLED", "true")

	suffix := uuid.NewString()[:8]
	supID := "sup_e2e_gl_" + suffix
	retID := "ret_e2e_gl_" + suffix
	ordID := "ord_e2e_gl_" + suffix
	now := time.Now().UTC().Add(-2 * time.Minute).Truncate(time.Microsecond)

	t.Cleanup(func() {
		cctx, ccancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer ccancel()
		for _, q := range []struct {
			sql, tbl string
			cols     []string
		}{
			{"SELECT EntityId, EntryId, Line FROM JournalPostings WHERE EntityId = @sid", "JournalPostings", []string{"EntityId", "EntryId", "Line"}},
			{"SELECT EntityId, EntryId FROM JournalEntries WHERE EntityId = @sid", "JournalEntries", []string{"EntityId", "EntryId"}},
			{"SELECT EntityId, PeriodId, AccountCode, PartyKey, Shard FROM GlBalances WHERE EntityId = @sid", "GlBalances", []string{"EntityId", "PeriodId", "AccountCode", "PartyKey", "Shard"}},
			{"SELECT EntryId FROM ArLedgerEntries WHERE SupplierId = @sid", "ArLedgerEntries", []string{"EntryId"}},
			{"SELECT InvoiceId FROM ArInvoices WHERE SupplierId = @sid", "ArInvoices", []string{"InvoiceId"}},
			{"SELECT EventId FROM OutboxEvents WHERE SupplierId = @sid", "OutboxEvents", []string{"EventId"}},
		} {
			var muts []*spanner.Mutation
			it := client.Single().Query(cctx, spanner.Statement{SQL: q.sql, Params: map[string]any{"sid": supID}})
			for {
				row, err := it.Next()
				if err != nil {
					break
				}
				vals := make([]*spanner.GenericColumnValue, len(q.cols))
				ptrs := make([]any, len(q.cols))
				for i := range q.cols {
					var sv spanner.GenericColumnValue
					vals[i] = &sv
					ptrs[i] = &sv
				}
				if row.Columns(ptrs...) == nil {
					k := make(spanner.Key, len(q.cols))
					for i, gv := range vals {
						if gv.Type.Code == 2 {
							var iv int64
							_ = gv.Decode(&iv)
							k[i] = iv
						} else {
							var s string
							_ = gv.Decode(&s)
							k[i] = s
						}
					}
					muts = append(muts, spanner.Delete(q.tbl, k))
				}
			}
			it.Stop()
			if len(muts) > 0 {
				_, _ = client.Apply(cctx, muts)
			}
		}
	})

	arSvc := ar.NewService(ar.NewSpannerRepository(client))
	arSvc.SetNow(func() time.Time { return now })
	inv, err := arSvc.OpenFromCreditLeave(context.Background(), ar.OpenFromCreditLeaveRequest{
		SupplierID: supID, RetailerID: retID, OrderID: ordID,
		AmountMinor: 95000, Currency: "UZS", CreditLeaveAt: now, TermsDays: 7,
	})
	if err != nil {
		t.Fatalf("OpenFromCreditLeave: %v", err)
	}
	if _, err := arSvc.ApplyPaymentFIFO(context.Background(), ar.PayFIFORequest{
		SupplierID:       supID,
		RetailerID:       retID,
		AmountMinor:      95000,
		Currency:         "UZS",
		PaymentMethod:    "BANK",
		IdempotencyKey:   "e2e-pay-" + suffix,
		EarmarkInvoiceID: inv.InvoiceID,
	}); err != nil {
		t.Fatalf("ApplyPaymentFIFO: %v", err)
	}

	admTok := h.IssueAdminToken(supID, "adm-e2e-live")
	periodID := gl.ComputePeriodID(context.Background(), supID, "UZ", now)
	tbResp := h.Client.Get("/v1/finance/gl/trial-balance?period="+periodID, admTok)
	if tbResp.StatusCode != http.StatusOK {
		t.Fatalf("HTTP trial-balance failed (%d): %s", tbResp.StatusCode, string(tbResp.Body))
	}
	var tb gl.TrialBalanceReport
	if err := json.Unmarshal(tbResp.Body, &tb); err != nil {
		t.Fatalf("decode trial balance: %v", err)
	}
	if !tb.Balanced || tb.TotalDebitsMinor != 95000 || tb.TotalCreditsMinor != 95000 {
		t.Fatalf("unexpected live TB via HTTP: %+v", tb)
	}

	// Cross-tenant admin isolation must fail closed (403).
	otherAdmTok := h.IssueToken(auth.RoleAdmin, "sup_other_"+suffix, "adm-other")
	isoResp := h.Client.Get("/v1/finance/gl/trial-balance?entity="+supID, otherAdmTok)
	if isoResp.StatusCode != http.StatusForbidden {
		t.Fatalf("cross-tenant trial-balance status = %d, want 403", isoResp.StatusCode)
	}

	chain, err := gl.VerifyEntityHashChain(context.Background(), client, supID)
	if err != nil || !chain.Valid || chain.EntriesCount != 1 {
		t.Fatalf("VerifyEntityHashChain got (%+v, %v), want Valid=true EntriesCount=1", chain, err)
	}
}

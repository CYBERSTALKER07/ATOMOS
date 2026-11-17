package order

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"google.golang.org/grpc/codes"
)

type draftTestRepo struct {
	Repository
	orders map[string]Order
	getErr error
}

func (r *draftTestRepo) GetOrder(_ context.Context, orderID string) (Order, bool, error) {
	if r.getErr != nil {
		return Order{}, false, r.getErr
	}
	ord, ok := r.orders[orderID]
	return ord, ok, nil
}

func newDraftTestService(orders map[string]Order) *Service {
	repo := &draftTestRepo{orders: orders}
	return newTestService(repo, time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC))
}

func invokeDraftHandler(t *testing.T, handler http.HandlerFunc, method string, body any, claims *auth.Claims, tenant *auth.TenantContext) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		if raw, ok := body.(string); ok {
			buf.WriteString(raw)
		} else if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatalf("encode request body: %v", err)
		}
	}
	req := httptest.NewRequest(method, "/v1/finance/drafts", &buf)
	req.Header.Set("Content-Type", "application/json")
	ctx := req.Context()
	if claims != nil {
		ctx = auth.WithClaims(ctx, *claims)
	}
	if tenant != nil {
		ctx = auth.WithTenant(ctx, *tenant)
	}
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()
	handler(rr, req)
	return rr
}

func TestResolveProposedAmountMinor(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		req       CreateDraftRequest
		draftType string
		want      int64
		wantErr   error
	}{
		{
			name:      "canonical int64 minor units large value",
			req:       CreateDraftRequest{ProposedAmountMinor: 900000000005},
			draftType: DraftTypeAdjustment,
			want:      900000000005,
		},
		{
			name:      "negative adjustment minor units allowed",
			req:       CreateDraftRequest{ProposedAmountMinor: -250000},
			draftType: DraftTypeAdjustment,
			want:      -250000,
		},
		{
			name:      "legacy alias accepted when canonical omitted",
			req:       CreateDraftRequest{ProposedAmount: 150075},
			draftType: DraftTypeAdjustment,
			want:      150075,
		},
		{
			name:      "matching canonical and alias accepted",
			req:       CreateDraftRequest{ProposedAmountMinor: 150075, ProposedAmount: 150075},
			draftType: DraftTypeCreditNote,
			want:      150075,
		},
		{
			name:      "conflicting canonical and alias rejected",
			req:       CreateDraftRequest{ProposedAmountMinor: 150075, ProposedAmount: 150000},
			draftType: DraftTypeAdjustment,
			wantErr:   ErrDraftValidation,
		},
		{
			name:      "zero amount rejected",
			req:       CreateDraftRequest{},
			draftType: DraftTypeAdjustment,
			wantErr:   ErrDraftValidation,
		},
		{
			name:      "negative credit note rejected",
			req:       CreateDraftRequest{ProposedAmountMinor: -100},
			draftType: DraftTypeCreditNote,
			wantErr:   ErrDraftValidation,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := resolveProposedAmountMinor(tc.req, tc.draftType)
			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("resolveProposedAmountMinor err = %v, want %v", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected err: %v", err)
			}
			if got != tc.want {
				t.Fatalf("amount = %d, want %d", got, tc.want)
			}
		})
	}
}

func TestValidateDraftReviewState_SoDAndIsolation(t *testing.T) {
	t.Parallel()

	t.Run("distinct reviewer within supplier allowed", func(t *testing.T) {
		t.Parallel()
		err := validateDraftReviewState("sup-1", "reviewer-2", "sup-1", "operator-1", draftStatusPendingApproval)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("self review rejected with SoD and forbidden sentinels", func(t *testing.T) {
		t.Parallel()
		err := validateDraftReviewState("sup-1", "  operator-1 ", "sup-1", "operator-1", draftStatusPendingApproval)
		if !errors.Is(err, ErrSelfReviewForbidden) {
			t.Fatalf("expected ErrSelfReviewForbidden, got %v", err)
		}
		if !errors.Is(err, ErrOrderForbidden) {
			t.Fatalf("expected ErrOrderForbidden, got %v", err)
		}
	})

	t.Run("empty operator rejected", func(t *testing.T) {
		t.Parallel()
		err := validateDraftReviewState("sup-1", "reviewer-2", "sup-1", "   ", draftStatusPendingApproval)
		if !errors.Is(err, ErrSelfReviewForbidden) || !errors.Is(err, ErrOrderForbidden) {
			t.Fatalf("expected ErrSelfReviewForbidden+ErrOrderForbidden, got %v", err)
		}
	})

	t.Run("empty reviewer rejected", func(t *testing.T) {
		t.Parallel()
		err := validateDraftReviewState("sup-1", "   ", "sup-1", "operator-1", draftStatusPendingApproval)
		if !errors.Is(err, ErrOrderForbidden) {
			t.Fatalf("expected ErrOrderForbidden, got %v", err)
		}
	})

	t.Run("cross-supplier review rejected", func(t *testing.T) {
		t.Parallel()
		err := validateDraftReviewState("sup-2", "reviewer-2", "sup-1", "operator-1", draftStatusPendingApproval)
		if !errors.Is(err, ErrOrderForbidden) {
			t.Fatalf("expected ErrOrderForbidden, got %v", err)
		}
	})

	t.Run("already reviewed draft returns conflict", func(t *testing.T) {
		t.Parallel()
		err := validateDraftReviewState("sup-1", "reviewer-2", "sup-1", "operator-1", draftStatusApproved)
		if !errors.Is(err, ErrDraftAlreadyReviewed) {
			t.Fatalf("expected ErrDraftAlreadyReviewed, got %v", err)
		}
	})
}

func TestValidateDraftCreateScope(t *testing.T) {
	t.Parallel()

	ord := Order{
		OrderID:    "ord-1",
		SupplierID: "sup-1",
		RetailerID: "ret-1",
		Status:     StatusCompleted,
	}

	t.Run("admin matching claims and tenant allowed", func(t *testing.T) {
		t.Parallel()
		ctx := auth.WithTenant(context.Background(), auth.TenantContext{SupplierID: "sup-1"})
		claims := auth.Claims{Subject: "op-1", Role: auth.RoleAdmin, SupplierID: "sup-1"}
		if err := validateDraftCreateScope(ctx, claims, ord); err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
	})

	t.Run("admin empty supplier fails closed", func(t *testing.T) {
		t.Parallel()
		claims := auth.Claims{Subject: "op-1", Role: auth.RoleAdmin}
		if err := validateDraftCreateScope(context.Background(), claims, ord); !errors.Is(err, ErrOrderForbidden) {
			t.Fatalf("expected ErrOrderForbidden, got %v", err)
		}
	})

	t.Run("admin mismatched tenant fails closed", func(t *testing.T) {
		t.Parallel()
		ctx := auth.WithTenant(context.Background(), auth.TenantContext{SupplierID: "sup-2"})
		claims := auth.Claims{Subject: "op-1", Role: auth.RoleAdmin, SupplierID: "sup-1"}
		if err := validateDraftCreateScope(ctx, claims, ord); !errors.Is(err, ErrOrderForbidden) {
			t.Fatalf("expected ErrOrderForbidden, got %v", err)
		}
	})

	t.Run("retailer matching org allowed", func(t *testing.T) {
		t.Parallel()
		claims := auth.Claims{Subject: "ret-user-1", Role: auth.RoleRetailer, RetailerOrgID: "ret-1"}
		if err := validateDraftCreateScope(context.Background(), claims, ord); err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
	})

	t.Run("retailer different org rejected", func(t *testing.T) {
		t.Parallel()
		claims := auth.Claims{Subject: "ret-user-2", Role: auth.RoleRetailer, RetailerOrgID: "ret-2"}
		if err := validateDraftCreateScope(context.Background(), claims, ord); !errors.Is(err, ErrOrderForbidden) {
			t.Fatalf("expected ErrOrderForbidden, got %v", err)
		}
	})

	t.Run("disallowed role rejected", func(t *testing.T) {
		t.Parallel()
		claims := auth.Claims{Subject: "wh-1", Role: auth.RoleWarehouseAdmin, SupplierID: "sup-1"}
		if err := validateDraftCreateScope(context.Background(), claims, ord); !errors.Is(err, ErrOrderForbidden) {
			t.Fatalf("expected ErrOrderForbidden, got %v", err)
		}
	})
}

func TestFinancialDrafts_ValidationAndFailClosed(t *testing.T) {
	t.Parallel()

	svc := newDraftTestService(map[string]Order{
		"ord-post-dispatch": {
			OrderID:    "ord-post-dispatch",
			SupplierID: "sup-1",
			RetailerID: "ret-1",
			Status:     StatusCompleted,
		},
		"ord-pre-dispatch": {
			OrderID:    "ord-pre-dispatch",
			SupplierID: "sup-1",
			RetailerID: "ret-1",
			Status:     StatusPending,
		},
	})

	validAdmin := auth.Claims{Subject: "admin-1", Role: auth.RoleAdmin, SupplierID: "sup-1"}

	t.Run("create rejects empty subject", func(t *testing.T) {
		_, err := svc.CreateFinancialDraft(context.Background(), auth.Claims{Role: auth.RoleAdmin, SupplierID: "sup-1"}, CreateDraftRequest{
			OrderID:             "ord-post-dispatch",
			ProposedAmountMinor: 5000,
			ReasonCode:          "DAMAGED",
		})
		if !errors.Is(err, ErrOrderForbidden) {
			t.Fatalf("expected ErrOrderForbidden, got %v", err)
		}
	})

	t.Run("create rejects zero amount", func(t *testing.T) {
		_, err := svc.CreateFinancialDraft(context.Background(), validAdmin, CreateDraftRequest{
			OrderID:    "ord-post-dispatch",
			ReasonCode: "DAMAGED",
		})
		if !errors.Is(err, ErrDraftValidation) {
			t.Fatalf("expected ErrDraftValidation, got %v", err)
		}
	})

	t.Run("create credit note endpoint rejects adjustment draftType", func(t *testing.T) {
		_, err := svc.CreateDraftCreditNote(context.Background(), validAdmin, CreateDraftRequest{
			OrderID:             "ord-post-dispatch",
			DraftType:           DraftTypeAdjustment,
			ProposedAmountMinor: 5000,
			ReasonCode:          "DAMAGED",
		})
		if !errors.Is(err, ErrDraftValidation) {
			t.Fatalf("expected ErrDraftValidation, got %v", err)
		}
	})

	t.Run("create fails closed when spannerClient is nil", func(t *testing.T) {
		_, err := svc.CreateFinancialDraft(context.Background(), validAdmin, CreateDraftRequest{
			OrderID:             "ord-post-dispatch",
			ProposedAmountMinor: 5000,
			ReasonCode:          "DAMAGED",
		})
		if !errors.Is(err, ErrFinancialDraftsUnavailable) {
			t.Fatalf("expected ErrFinancialDraftsUnavailable, got %v", err)
		}
	})

	t.Run("review rejects non-admin, empty subject, empty supplier, and nil spannerClient", func(t *testing.T) {
		_, err := svc.ReviewFinancialDraft(context.Background(), auth.Claims{Subject: "ret-1", Role: auth.RoleRetailer}, ReviewDraftRequest{
			DraftID: "d-1", Action: "APPROVE",
		})
		if !errors.Is(err, ErrOrderForbidden) {
			t.Fatalf("non-admin: expected ErrOrderForbidden, got %v", err)
		}

		_, err = svc.ReviewFinancialDraft(context.Background(), auth.Claims{Role: auth.RoleAdmin, SupplierID: "sup-1"}, ReviewDraftRequest{
			DraftID: "d-1", Action: "APPROVE",
		})
		if !errors.Is(err, ErrOrderForbidden) {
			t.Fatalf("empty subject: expected ErrOrderForbidden, got %v", err)
		}

		_, err = svc.ReviewFinancialDraft(context.Background(), auth.Claims{Subject: "admin-2", Role: auth.RoleAdmin}, ReviewDraftRequest{
			DraftID: "d-1", Action: "APPROVE",
		})
		if !errors.Is(err, ErrOrderForbidden) {
			t.Fatalf("empty supplier: expected ErrOrderForbidden, got %v", err)
		}

		_, err = svc.ReviewDraftCreditNote(context.Background(), validAdmin, ReviewDraftRequest{
			DraftID: "d-1", Action: "APPROVE",
		})
		if !errors.Is(err, ErrFinancialDraftsUnavailable) {
			t.Fatalf("nil client: expected ErrFinancialDraftsUnavailable, got %v", err)
		}
	})
}

func TestFinancialDraftHTTPHandlers_StatusMapping(t *testing.T) {
	t.Parallel()

	svc := newDraftTestService(map[string]Order{
		"ord-post-dispatch": {
			OrderID:    "ord-post-dispatch",
			SupplierID: "sup-1",
			RetailerID: "ret-1",
			Status:     StatusCompleted,
		},
	})

	adminClaims := &auth.Claims{Subject: "admin-1", Role: auth.RoleAdmin, SupplierID: "sup-1"}

	if rr := invokeDraftHandler(t, svc.HandleCreateFinancialDraft, http.MethodGet, nil, adminClaims, nil); rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("method check = %d, want 405", rr.Code)
	}
	if rr := invokeDraftHandler(t, svc.HandleCreateFinancialDraft, http.MethodPost, nil, nil, nil); rr.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated = %d, want 401", rr.Code)
	}
	if rr := invokeDraftHandler(t, svc.HandleCreateFinancialDraft, http.MethodPost, "{bad", adminClaims, nil); rr.Code != http.StatusBadRequest {
		t.Fatalf("bad json = %d, want 400", rr.Code)
	}
	if rr := invokeDraftHandler(t, svc.HandleCreateFinancialDraft, http.MethodPost, CreateDraftRequest{
		OrderID: "ord-post-dispatch", ReasonCode: "DAMAGED",
	}, adminClaims, nil); rr.Code != http.StatusBadRequest {
		t.Fatalf("zero amount = %d, want 400", rr.Code)
	}
	if rr := invokeDraftHandler(t, svc.HandleCreateFinancialDraft, http.MethodPost, CreateDraftRequest{
		OrderID: "ord-post-dispatch", ProposedAmountMinor: 125000, ReasonCode: "DAMAGED",
	}, adminClaims, nil); rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("nil spanner client = %d, want 503", rr.Code)
	}

	// Verify writeFinancialDraftError HTTP codes for each sentinel.
	errorStatusCases := []struct {
		name string
		err  error
		want int
	}{
		{name: "validation", err: ErrDraftValidation, want: http.StatusBadRequest},
		{name: "self-review SoD", err: draftSentinelError{primary: ErrSelfReviewForbidden, secondary: ErrOrderForbidden}, want: http.StatusForbidden},
		{name: "forbidden", err: ErrOrderForbidden, want: http.StatusForbidden},
		{name: "not found", err: ErrDraftNotFound, want: http.StatusNotFound},
		{name: "conflict", err: ErrDraftAlreadyReviewed, want: http.StatusConflict},
		{name: "unavailable", err: ErrFinancialDraftsUnavailable, want: http.StatusServiceUnavailable},
	}
	for _, tc := range errorStatusCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			rr := httptest.NewRecorder()
			svc.writeFinancialDraftError(rr, "test", "id-1", tc.err)
			if rr.Code != tc.want {
				t.Fatalf("status = %d, want %d (body=%s)", rr.Code, tc.want, rr.Body.String())
			}
		})
	}
}

// --- Live Spanner Emulator Integration Tests ---

func requireDraftSpannerClient(t *testing.T, ctx context.Context) *spanner.Client {
	t.Helper()
	client := newSpannerIntegrationClient(t, ctx)
	probeCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	_, err := client.Single().ReadRow(probeCtx, draftTableAdjustments, spanner.Key{"__t9_probe__"}, []string{"DraftId"})
	if err != nil && !isSpannerNotFound(err) {
		code := spanner.ErrCode(err)
		if code == codes.Unavailable || code == codes.DeadlineExceeded || errors.Is(err, context.DeadlineExceeded) {
			t.Skipf("spanner emulator unreachable at %q: %v", os.Getenv("SPANNER_EMULATOR_HOST"), err)
		}
		t.Fatalf("probe DraftAdjustments on live emulator: %v", err)
	}
	return client
}

func cleanupSpannerDraftRow(t *testing.T, client *spanner.Client, tableName, draftID string) {
	t.Helper()
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_, _ = client.Apply(ctx, []*spanner.Mutation{
			spanner.Delete(tableName, spanner.Key{draftID}),
		})
	})
}

func TestSpannerDraftAdjustments_MinorUnitsAndSoD(t *testing.T) {
	ctx := context.Background()
	client := requireDraftSpannerClient(t, ctx)

	supplierA := "sup-t9-a-" + uuid.NewString()[:8]
	supplierB := "sup-t9-b-" + uuid.NewString()[:8]
	orderID := "ord-t9-" + uuid.NewString()[:8]
	operatorID := "op-t9-" + uuid.NewString()[:8]
	reviewerID := "rev-t9-" + uuid.NewString()[:8]
	otherAdminID := "rev2-t9-" + uuid.NewString()[:8]

	svc := newDraftTestService(map[string]Order{
		orderID: {
			OrderID:    orderID,
			SupplierID: supplierA,
			RetailerID: "ret-t9-" + uuid.NewString()[:8],
			Status:     StatusCompleted,
		},
	})
	svc.spannerClient = client

	opClaims := &auth.Claims{Subject: operatorID, Role: auth.RoleAdmin, SupplierID: supplierA}
	opTenant := &auth.TenantContext{SupplierID: supplierA}
	revClaims := &auth.Claims{Subject: reviewerID, Role: auth.RoleAdmin, SupplierID: supplierA}
	revTenant := &auth.TenantContext{SupplierID: supplierA}
	otherSupplierClaims := &auth.Claims{Subject: reviewerID, Role: auth.RoleAdmin, SupplierID: supplierB}
	otherSupplierTenant := &auth.TenantContext{SupplierID: supplierB}
	otherAdminClaims := &auth.Claims{Subject: otherAdminID, Role: auth.RoleAdmin, SupplierID: supplierA}

	// 1. Missing draft returns 404 (validates spanner ReadRow NotFound mapping).
	rrNotFound := invokeDraftHandler(t, svc.HandleReviewFinancialDraft, http.MethodPost, ReviewDraftRequest{
		DraftID: "missing-" + uuid.NewString(),
		Action:  "APPROVE",
	}, revClaims, revTenant)
	if rrNotFound.Code != http.StatusNotFound {
		t.Fatalf("missing draft status = %d, want 404 (body=%s)", rrNotFound.Code, rrNotFound.Body.String())
	}

	// 2. Create adjustment draft with >32-bit int64 minor units.
	const wantMinor int64 = 900000000005
	rrCreate := invokeDraftHandler(t, svc.HandleCreateFinancialDraft, http.MethodPost, CreateDraftRequest{
		OrderID:             orderID,
		ProposedAmountMinor: wantMinor,
		ReasonCode:          "SHORT_DELIVERY",
		Notes:               "T9 minor units test",
	}, opClaims, opTenant)
	if rrCreate.Code != http.StatusCreated {
		t.Fatalf("create draft status = %d, want 201 (body=%s)", rrCreate.Code, rrCreate.Body.String())
	}
	var created DraftResponse
	if err := json.NewDecoder(rrCreate.Body).Decode(&created); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	if created.DraftID == "" || created.Status != draftStatusPendingApproval || created.ProposedAmountMinor != wantMinor {
		t.Fatalf("unexpected create response: %+v", created)
	}
	cleanupSpannerDraftRow(t, client, draftTableAdjustments, created.DraftID)

	// 3. Self-approval and self-rejection must fail with 403 Forbidden (SoD).
	for _, action := range []string{"APPROVE", "REJECT"} {
		rrSelf := invokeDraftHandler(t, svc.HandleReviewFinancialDraft, http.MethodPost, ReviewDraftRequest{
			DraftID: created.DraftID,
			Action:  action,
		}, opClaims, opTenant)
		if rrSelf.Code != http.StatusForbidden {
			t.Fatalf("self-%s status = %d, want 403 (body=%s)", action, rrSelf.Code, rrSelf.Body.String())
		}
	}

	// 4. Cross-supplier admin must fail with 403 Forbidden.
	rrCross := invokeDraftHandler(t, svc.HandleReviewFinancialDraft, http.MethodPost, ReviewDraftRequest{
		DraftID: created.DraftID,
		Action:  "APPROVE",
	}, otherSupplierClaims, otherSupplierTenant)
	if rrCross.Code != http.StatusForbidden {
		t.Fatalf("cross-supplier review status = %d, want 403 (body=%s)", rrCross.Code, rrCross.Body.String())
	}

	// 5. Distinct same-supplier admin succeeds with 200 OK.
	rrApprove := invokeDraftHandler(t, svc.HandleReviewFinancialDraft, http.MethodPost, ReviewDraftRequest{
		DraftID: created.DraftID,
		Action:  "APPROVE",
	}, revClaims, revTenant)
	if rrApprove.Code != http.StatusOK {
		t.Fatalf("distinct reviewer approve status = %d, want 200 (body=%s)", rrApprove.Code, rrApprove.Body.String())
	}
	var approved DraftResponse
	if err := json.NewDecoder(rrApprove.Body).Decode(&approved); err != nil {
		t.Fatalf("decode approve response: %v", err)
	}
	if approved.Status != draftStatusApproved || approved.ProposedAmountMinor != wantMinor {
		t.Fatalf("unexpected approve response: %+v", approved)
	}

	// Verify persisted Spanner columns (INT64 minor units + audit fields).
	row, err := client.Single().ReadRow(ctx, draftTableAdjustments, spanner.Key{created.DraftID}, []string{
		"ProposedAmountMinor", "OperatorId", "Status", "ReviewedBy", "ReviewedAt",
	})
	if err != nil {
		t.Fatalf("read updated DraftAdjustments row: %v", err)
	}
	var gotMinor int64
	var gotOperator, gotStatus string
	var gotReviewedBy spanner.NullString
	var gotReviewedAt spanner.NullTime
	if err := row.Columns(&gotMinor, &gotOperator, &gotStatus, &gotReviewedBy, &gotReviewedAt); err != nil {
		t.Fatalf("scan DraftAdjustments row: %v", err)
	}
	if gotMinor != wantMinor || gotOperator != operatorID || gotStatus != draftStatusApproved || !gotReviewedBy.Valid || gotReviewedBy.StringVal != reviewerID || !gotReviewedAt.Valid {
		t.Fatalf("unexpected persisted adjustment row: minor=%d op=%q status=%q revBy=%v revAt=%v", gotMinor, gotOperator, gotStatus, gotReviewedBy, gotReviewedAt)
	}

	// 6. Second review attempt returns 409 Conflict.
	rrDup := invokeDraftHandler(t, svc.HandleReviewFinancialDraft, http.MethodPost, ReviewDraftRequest{
		DraftID: created.DraftID,
		Action:  "REJECT",
	}, otherAdminClaims, revTenant)
	if rrDup.Code != http.StatusConflict {
		t.Fatalf("duplicate review status = %d, want 409 (body=%s)", rrDup.Code, rrDup.Body.String())
	}
}

func TestSpannerDraftCreditNotes_MinorUnitsAndSoD(t *testing.T) {
	ctx := context.Background()
	client := requireDraftSpannerClient(t, ctx)

	supplierA := "sup-cn-a-" + uuid.NewString()[:8]
	supplierB := "sup-cn-b-" + uuid.NewString()[:8]
	retailerA := "ret-cn-a-" + uuid.NewString()[:8]
	orderID := "ord-cn-" + uuid.NewString()[:8]
	operatorAdminID := "op-cn-" + uuid.NewString()[:8]
	reviewerAdminID := "rev-cn-" + uuid.NewString()[:8]

	svc := newDraftTestService(map[string]Order{
		orderID: {
			OrderID:    orderID,
			SupplierID: supplierA,
			RetailerID: retailerA,
			Status:     StatusCompleted,
		},
	})
	svc.spannerClient = client

	opClaims := &auth.Claims{Subject: operatorAdminID, Role: auth.RoleAdmin, SupplierID: supplierA}
	opTenant := &auth.TenantContext{SupplierID: supplierA}
	revClaims := &auth.Claims{Subject: reviewerAdminID, Role: auth.RoleAdmin, SupplierID: supplierA}
	revTenant := &auth.TenantContext{SupplierID: supplierA}
	crossClaims := &auth.Claims{Subject: reviewerAdminID, Role: auth.RoleAdmin, SupplierID: supplierB}
	crossTenant := &auth.TenantContext{SupplierID: supplierB}

	const wantMinor int64 = 250050
	rrCreate := invokeDraftHandler(t, svc.HandleCreateDraftCreditNote, http.MethodPost, CreateDraftRequest{
		OrderID:             orderID,
		ProposedAmountMinor: wantMinor,
		ReasonCode:          "RETURN_DAMAGED",
		Notes:               "Credit note draft T9",
	}, opClaims, opTenant)
	if rrCreate.Code != http.StatusCreated {
		t.Fatalf("create credit note draft status = %d, want 201 (body=%s)", rrCreate.Code, rrCreate.Body.String())
	}
	var created DraftResponse
	if err := json.NewDecoder(rrCreate.Body).Decode(&created); err != nil {
		t.Fatalf("decode create credit note response: %v", err)
	}
	if created.DraftID == "" || created.DraftType != DraftTypeCreditNote || created.ProposedAmountMinor != wantMinor {
		t.Fatalf("unexpected credit note response: %+v", created)
	}
	cleanupSpannerDraftRow(t, client, draftTableCreditNotes, created.DraftID)

	// Self-review must fail with 403 on BOTH the dedicated credit-note endpoint AND the generic review endpoint.
	rrSelfCN := invokeDraftHandler(t, svc.HandleReviewDraftCreditNote, http.MethodPost, ReviewDraftRequest{
		DraftID: created.DraftID,
		Action:  "APPROVE",
	}, opClaims, opTenant)
	if rrSelfCN.Code != http.StatusForbidden {
		t.Fatalf("credit-note self-review status = %d, want 403 (body=%s)", rrSelfCN.Code, rrSelfCN.Body.String())
	}

	rrSelfGeneric := invokeDraftHandler(t, svc.HandleReviewFinancialDraft, http.MethodPost, ReviewDraftRequest{
		DraftID: created.DraftID,
		Action:  "APPROVE",
	}, opClaims, opTenant)
	if rrSelfGeneric.Code != http.StatusForbidden {
		t.Fatalf("generic self-review on credit-note status = %d, want 403 (body=%s)", rrSelfGeneric.Code, rrSelfGeneric.Body.String())
	}

	// Cross-supplier review fails with 403.
	rrCross := invokeDraftHandler(t, svc.HandleReviewDraftCreditNote, http.MethodPost, ReviewDraftRequest{
		DraftID: created.DraftID,
		Action:  "APPROVE",
	}, crossClaims, crossTenant)
	if rrCross.Code != http.StatusForbidden {
		t.Fatalf("cross-supplier credit-note review status = %d, want 403 (body=%s)", rrCross.Code, rrCross.Body.String())
	}

	// Distinct same-supplier reviewer rejects cleanly (200 OK).
	rrReject := invokeDraftHandler(t, svc.HandleReviewDraftCreditNote, http.MethodPost, ReviewDraftRequest{
		DraftID: created.DraftID,
		Action:  "REJECT",
	}, revClaims, revTenant)
	if rrReject.Code != http.StatusOK {
		t.Fatalf("distinct reviewer credit-note reject status = %d, want 200 (body=%s)", rrReject.Code, rrReject.Body.String())
	}

	// Verify persisted Spanner columns on DraftCreditNotes.
	row, err := client.Single().ReadRow(ctx, draftTableCreditNotes, spanner.Key{created.DraftID}, []string{
		"RetailerId", "ProposedAmountMinor", "OperatorId", "Status", "ReviewedBy", "ReviewedAt",
	})
	if err != nil {
		t.Fatalf("read DraftCreditNotes row: %v", err)
	}
	var gotRetailer, gotOperator, gotStatus string
	var gotMinor int64
	var gotReviewedBy spanner.NullString
	var gotReviewedAt spanner.NullTime
	if err := row.Columns(&gotRetailer, &gotMinor, &gotOperator, &gotStatus, &gotReviewedBy, &gotReviewedAt); err != nil {
		t.Fatalf("scan DraftCreditNotes row: %v", err)
	}
	if gotRetailer != retailerA || gotMinor != wantMinor || gotOperator != operatorAdminID || gotStatus != draftStatusRejected || !gotReviewedBy.Valid || gotReviewedBy.StringVal != reviewerAdminID || !gotReviewedAt.Valid {
		t.Fatalf("unexpected DraftCreditNotes row: retailer=%q minor=%d op=%q status=%q revBy=%v revAt=%v", gotRetailer, gotMinor, gotOperator, gotStatus, gotReviewedBy, gotReviewedAt)
	}

	// Duplicate review returns 409 Conflict.
	rrDup := invokeDraftHandler(t, svc.HandleReviewDraftCreditNote, http.MethodPost, ReviewDraftRequest{
		DraftID: created.DraftID,
		Action:  "APPROVE",
	}, revClaims, revTenant)
	if rrDup.Code != http.StatusConflict {
		t.Fatalf("duplicate credit-note review status = %d, want 409 (body=%s)", rrDup.Code, rrDup.Body.String())
	}
}

func TestFinancialDrafts_SupplierRBAC_AuditorAgentAndPermEnforcement(t *testing.T) {
	t.Parallel()

	svc := newDraftTestService(map[string]Order{
		"ord-1": {OrderID: "ord-1", SupplierID: "sup-1", RetailerID: "ret-1", Status: StatusCompleted},
	})

	var sodCalls []DraftSoDCheckInput
	svc.SetDraftSoDCheck(func(_ context.Context, in DraftSoDCheckInput) error {
		sodCalls = append(sodCalls, in)
		return nil
	})

	auditorClaims := auth.Claims{Subject: "aud-1", Role: auth.RoleAdmin, SupplierRole: auth.SupplierRoleAuditor, SupplierID: "sup-1"}
	if _, err := svc.CreateFinancialDraft(context.Background(), auditorClaims, CreateDraftRequest{
		OrderID: "ord-1", ProposedAmountMinor: 1000, ReasonCode: "DAMAGED",
	}); !errors.Is(err, ErrOrderForbidden) {
		t.Fatalf("auditor create draft err=%v want ErrOrderForbidden", err)
	}

	if _, err := svc.ReviewFinancialDraft(context.Background(), auditorClaims, ReviewDraftRequest{
		DraftID: "draft-1", Action: "APPROVE",
	}); !errors.Is(err, ErrOrderForbidden) {
		t.Fatalf("auditor review draft err=%v want ErrOrderForbidden", err)
	}

	agentClaims := auth.Claims{Subject: "agent-1", Role: auth.RoleAdmin, ActorType: auth.ActorTypeAgent, SupplierRole: auth.SupplierRoleAgent, SupplierID: "sup-1"}
	if _, err := svc.ReviewDraftCreditNote(context.Background(), agentClaims, ReviewDraftRequest{
		DraftID: "draft-1", Action: "APPROVE",
	}); !errors.Is(err, ErrOrderForbidden) {
		t.Fatalf("agent review credit note err=%v want ErrOrderForbidden", err)
	}

	accountantClaims := auth.Claims{Subject: "acct-1", Role: auth.RoleAdmin, SupplierRole: auth.SupplierRoleAccountant, SupplierID: "sup-1"}
	if _, err := svc.ReviewDraftCreditNote(context.Background(), accountantClaims, ReviewDraftRequest{
		DraftID: "draft-1", Action: "APPROVE",
	}); !errors.Is(err, ErrOrderForbidden) {
		t.Fatalf("accountant lacking credit_note.approve review err=%v want ErrOrderForbidden", err)
	}

	warehouseClaims := auth.Claims{Subject: "wh-1", Role: auth.RoleAdmin, SupplierRole: auth.SupplierRoleWarehouseManager, SupplierID: "sup-1"}
	if _, err := svc.CreateDraftCreditNote(context.Background(), warehouseClaims, CreateDraftRequest{
		OrderID: "ord-1", ProposedAmountMinor: 1000, ReasonCode: "DAMAGED",
	}); !errors.Is(err, ErrOrderForbidden) {
		t.Fatalf("warehouse manager lacking credit_note.propose err=%v want ErrOrderForbidden", err)
	}

	// AUDITOR and AGENT review attempts should trigger draftSoDCheck hook for violation auditing.
	if len(sodCalls) < 2 {
		t.Fatalf("expected SoD check hook calls for auditor/agent attempts, got %d", len(sodCalls))
	}
}

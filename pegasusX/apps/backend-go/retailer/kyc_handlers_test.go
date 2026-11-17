package retailer

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

type mockKycRepo struct {
	testRetailerRepo
	docs                   map[string]KycDocument
	updatedDocID           string
	updatedStatus          string
	updatedReviewedBy      string
	updatedRejectionReason string
}

func (m *mockKycRepo) GetKycDocument(_ context.Context, docID string) (KycDocument, bool, error) {
	if m.docs != nil {
		doc, ok := m.docs[docID]
		return doc, ok, nil
	}
	return KycDocument{DocumentID: docID, RetailerID: "ret-default"}, true, nil
}

func (m *mockKycRepo) GetRetailer(ctx context.Context, id string) (Retailer, bool, error) {
	if m.testRetailerRepo.retailer.SupplierID != "" {
		return m.testRetailerRepo.GetRetailer(ctx, id)
	}
	return Retailer{RetailerID: id, SupplierID: "sup-1"}, true, nil
}

func (m *mockKycRepo) UpdateKycDocumentStatus(_ context.Context, docID string, status string, reviewedBy string, rejectionReason string) error {
	m.updatedDocID = docID
	m.updatedStatus = status
	m.updatedReviewedBy = reviewedBy
	m.updatedRejectionReason = rejectionReason
	return nil
}

func TestHandleReviewKyc_Unauthenticated(t *testing.T) {
	svc := NewService(ServiceConfig{Repo: &mockKycRepo{}})

	req := httptest.NewRequest(http.MethodPost, "/v1/admin/retailer-kyc/doc-1/review", strings.NewReader(`{"status":"APPROVED"}`))
	rr := httptest.NewRecorder()

	svc.HandleReviewKyc(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d want=401 body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleReviewKyc_ForbiddenForRetailer(t *testing.T) {
	svc := NewService(ServiceConfig{Repo: &mockKycRepo{}})

	req := httptest.NewRequest(http.MethodPost, "/v1/admin/retailer-kyc/doc-1/review", strings.NewReader(`{"status":"APPROVED"}`))
	ctx := auth.WithClaims(req.Context(), auth.Claims{
		Subject: "ret-user-1",
		Role:    auth.RoleRetailer,
	})
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	svc.HandleReviewKyc(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("status=%d want=403 body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleReviewKyc_AdminSuccess_DerivesSubject(t *testing.T) {
	repo := &mockKycRepo{}
	svc := NewService(ServiceConfig{Repo: repo})

	r := chi.NewRouter()
	r.Post("/v1/admin/retailer-kyc/{documentID}/review", svc.HandleReviewKyc)

	req := httptest.NewRequest(http.MethodPost, "/v1/admin/retailer-kyc/doc-xyz/review", strings.NewReader(`{"status":"APPROVED"}`))
	ctx := auth.WithClaims(req.Context(), auth.Claims{
		Subject:    "admin-sec-99",
		Role:       auth.RoleAdmin,
		SupplierID: "sup-1",
	})
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d want=200 body=%s", rr.Code, rr.Body.String())
	}

	var resp map[string]string
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp["status"] != "updated" {
		t.Fatalf("unexpected status: %v", resp)
	}

	if repo.updatedDocID != "doc-xyz" {
		t.Fatalf("docID=%q want doc-xyz", repo.updatedDocID)
	}
	if repo.updatedStatus != "APPROVED" {
		t.Fatalf("status=%q want APPROVED", repo.updatedStatus)
	}
	if repo.updatedReviewedBy != "admin-sec-99" {
		t.Fatalf("reviewedBy=%q want admin-sec-99", repo.updatedReviewedBy)
	}
}

func TestHandleReviewKyc_PlatformAdminSuccess(t *testing.T) {
	repo := &mockKycRepo{}
	svc := NewService(ServiceConfig{Repo: repo})

	r := chi.NewRouter()
	r.Post("/v1/admin/retailer-kyc/{documentID}/review", svc.HandleReviewKyc)

	req := httptest.NewRequest(http.MethodPost, "/v1/admin/retailer-kyc/doc-rej/review", strings.NewReader(`{"status":"REJECTED","rejection_reason":"blurry"}`))
	ctx := auth.WithClaims(req.Context(), auth.Claims{
		Subject: "platform-admin-7",
		Role:    auth.RolePlatformAdmin,
	})
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d want=200 body=%s", rr.Code, rr.Body.String())
	}

	if repo.updatedDocID != "doc-rej" {
		t.Fatalf("docID=%q want doc-rej", repo.updatedDocID)
	}
	if repo.updatedStatus != "REJECTED" {
		t.Fatalf("status=%q want REJECTED", repo.updatedStatus)
	}
	if repo.updatedReviewedBy != "platform-admin-7" {
		t.Fatalf("reviewedBy=%q want platform-admin-7", repo.updatedReviewedBy)
	}
	if repo.updatedRejectionReason != "blurry" {
		t.Fatalf("rejectionReason=%q want blurry", repo.updatedRejectionReason)
	}
}

func TestHandleReviewKyc_InvalidStatuses_Rejected(t *testing.T) {
	invalidStatuses := []string{
		"UNKNOWN_STATUS",
		"",
		"INVALID",
		"PENDING",
		"HACKED",
		"APPROVED_EXTRA",
		"REJECTED_DOC",
		"   ",
		"random_string",
		"12345",
	}

	for _, invalidStatus := range invalidStatuses {
		t.Run("status_"+invalidStatus, func(t *testing.T) {
			repo := &mockKycRepo{}
			svc := NewService(ServiceConfig{Repo: repo})

			r := chi.NewRouter()
			r.Post("/v1/admin/retailer-kyc/{documentID}/review", svc.HandleReviewKyc)

			bodyJSON, err := json.Marshal(map[string]string{
				"status":           invalidStatus,
				"rejection_reason": "irrelevant",
			})
			if err != nil {
				t.Fatalf("marshal error: %v", err)
			}

			req := httptest.NewRequest(http.MethodPost, "/v1/admin/retailer-kyc/doc-invalid/review", strings.NewReader(string(bodyJSON)))
			ctx := auth.WithClaims(req.Context(), auth.Claims{
				Subject:    "admin-1",
				Role:       auth.RoleAdmin,
				SupplierID: "sup-1",
			})
			req = req.WithContext(ctx)
			rr := httptest.NewRecorder()

			r.ServeHTTP(rr, req)

			if rr.Code != http.StatusBadRequest {
				t.Fatalf("status=%d want=400 body=%s", rr.Code, rr.Body.String())
			}

			var resp map[string]string
			if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
				t.Fatalf("decode: %v", err)
			}
			if resp["error"] != "invalid_status" {
				t.Fatalf("error=%q want invalid_status", resp["error"])
			}

			// Ensure repository was NOT called
			if repo.updatedDocID != "" {
				t.Fatalf("UpdateKycDocumentStatus called with docID=%q, want empty", repo.updatedDocID)
			}
			if repo.updatedStatus != "" {
				t.Fatalf("UpdateKycDocumentStatus called with status=%q, want empty", repo.updatedStatus)
			}
		})
	}
}

func TestHandleReviewKyc_ValidStatuses_CaseInsensitive(t *testing.T) {
	cases := []struct {
		inputStatus    string
		expectedStatus string
	}{
		{"APPROVED", "APPROVED"},
		{"REJECTED", "REJECTED"},
		{"approved", "APPROVED"},
		{"rejected", "REJECTED"},
		{"  APPROVED  ", "APPROVED"},
		{"  rejected  ", "REJECTED"},
		{"ApPrOvEd", "APPROVED"},
		{"ReJeCtEd", "REJECTED"},
	}

	for _, tc := range cases {
		t.Run("status_"+tc.inputStatus, func(t *testing.T) {
			repo := &mockKycRepo{}
			svc := NewService(ServiceConfig{Repo: repo})

			r := chi.NewRouter()
			r.Post("/v1/admin/retailer-kyc/{documentID}/review", svc.HandleReviewKyc)

			bodyJSON, err := json.Marshal(map[string]string{
				"status":           tc.inputStatus,
				"rejection_reason": "valid reason",
			})
			if err != nil {
				t.Fatalf("marshal error: %v", err)
			}

			req := httptest.NewRequest(http.MethodPost, "/v1/admin/retailer-kyc/doc-valid-case/review", strings.NewReader(string(bodyJSON)))
			ctx := auth.WithClaims(req.Context(), auth.Claims{
				Subject:    "admin-validator",
				Role:       auth.RoleAdmin,
				SupplierID: "sup-1",
			})
			req = req.WithContext(ctx)
			rr := httptest.NewRecorder()

			r.ServeHTTP(rr, req)

			if rr.Code != http.StatusOK {
				t.Fatalf("status=%d want=200 body=%s", rr.Code, rr.Body.String())
			}

			var resp map[string]string
			if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
				t.Fatalf("decode: %v", err)
			}
			if resp["status"] != "updated" {
				t.Fatalf("resp[status]=%q want updated", resp["status"])
			}

			// Ensure repository WAS called with normalized uppercase status
			if repo.updatedDocID != "doc-valid-case" {
				t.Fatalf("docID=%q want doc-valid-case", repo.updatedDocID)
			}
			if repo.updatedStatus != tc.expectedStatus {
				t.Fatalf("status=%q want %q", repo.updatedStatus, tc.expectedStatus)
			}
			if repo.updatedReviewedBy != "admin-validator" {
				t.Fatalf("reviewedBy=%q want admin-validator", repo.updatedReviewedBy)
			}
			if repo.updatedRejectionReason != "valid reason" {
				t.Fatalf("rejectionReason=%q want valid reason", repo.updatedRejectionReason)
			}
		})
	}
}

func TestHandleReviewKyc_SupplierAdmin_Authorization(t *testing.T) {
	repo := &mockKycRepo{
		docs: map[string]KycDocument{
			"doc-ret-1": {DocumentID: "doc-ret-1", RetailerID: "ret-1"},
		},
		testRetailerRepo: testRetailerRepo{
			found:    true,
			retailer: Retailer{RetailerID: "ret-1", SupplierID: "sup-authorized"},
		},
	}
	svc := NewService(ServiceConfig{Repo: repo})

	r := chi.NewRouter()
	r.Post("/v1/admin/retailer-kyc/{documentID}/review", svc.HandleReviewKyc)

	// Case 1: Supplier Admin without SupplierID -> 403 Forbidden
	{
		req := httptest.NewRequest(http.MethodPost, "/v1/admin/retailer-kyc/doc-ret-1/review", strings.NewReader(`{"status":"APPROVED"}`))
		req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{
			Subject: "admin-no-sup",
			Role:    auth.RoleAdmin,
		}))
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusForbidden {
			t.Fatalf("expected 403 when SupplierID is missing, got %d body=%s", rr.Code, rr.Body.String())
		}
	}

	// Case 2: Unlinked/competing Supplier Admin -> 403 Forbidden
	{
		req := httptest.NewRequest(http.MethodPost, "/v1/admin/retailer-kyc/doc-ret-1/review", strings.NewReader(`{"status":"APPROVED"}`))
		req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{
			Subject:    "admin-competing",
			Role:       auth.RoleAdmin,
			SupplierID: "sup-competitor",
		}))
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusForbidden {
			t.Fatalf("expected 403 for unlinked competitor supplier, got %d body=%s", rr.Code, rr.Body.String())
		}
	}

	// Case 3: Linked Supplier Admin -> 200 OK
	{
		req := httptest.NewRequest(http.MethodPost, "/v1/admin/retailer-kyc/doc-ret-1/review", strings.NewReader(`{"status":"APPROVED"}`))
		req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{
			Subject:    "admin-authorized",
			Role:       auth.RoleAdmin,
			SupplierID: "sup-authorized",
		}))
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200 for linked supplier admin, got %d body=%s", rr.Code, rr.Body.String())
		}
		if repo.updatedStatus != "APPROVED" {
			t.Fatalf("expected status APPROVED, got %q", repo.updatedStatus)
		}
	}

	// Case 4: Platform Admin -> 200 OK (retains global authority, even without SupplierID)
	{
		req := httptest.NewRequest(http.MethodPost, "/v1/admin/retailer-kyc/doc-ret-1/review", strings.NewReader(`{"status":"APPROVED"}`))
		req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{
			Subject: "platform-global-admin",
			Role:    auth.RolePlatformAdmin,
		}))
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200 for platform admin global review authority, got %d body=%s", rr.Code, rr.Body.String())
		}
	}
}

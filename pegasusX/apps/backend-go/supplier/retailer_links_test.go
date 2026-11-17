package supplier

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/entityresolution"
	"github.com/pegasusx/pegasusx/apps/backend-go/partner"
)

func supplierAdminReq(method, target, body, supplierID string) *http.Request {
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	ctx := auth.WithClaims(req.Context(), auth.Claims{
		Subject:    "admin-" + supplierID,
		Role:       auth.RoleAdmin,
		SupplierID: supplierID,
	})
	return req.WithContext(ctx)
}

func TestHandleRetailerLinks_AndDedup_TenantIsolated(t *testing.T) {
	now := time.Date(2026, 11, 13, 9, 0, 0, 0, time.UTC)
	svc := NewService(ServiceConfig{
		Repo:       &pricingTestRepo{},
		SupplierID: "sup-1",
		Now:        func() time.Time { return now },
	})

	r := chi.NewRouter()
	r.Get("/v1/supplier/retailer-links", svc.HandleRetailerLinks)
	r.Post("/v1/supplier/retailer-links", svc.HandleRetailerLinks)
	r.Get("/v1/supplier/retailer-links/dedup", svc.HandleRetailerLinksDedup)
	r.Post("/v1/supplier/retailer-links/{retailerID}/resolve", svc.HandleResolveRetailerLink)

	// 1) Upsert first retailer for sup-1.
	rr1 := httptest.NewRecorder()
	r.ServeHTTP(rr1, supplierAdminReq(http.MethodPost, "/v1/supplier/retailer-links",
		`{"retailer_id":"ret-1","tax_id_tin":"INN 301-234 567","phone":"90 111 22 33","legal_name":"Baraka"}`, "sup-1"))
	if rr1.Code != http.StatusOK {
		t.Fatalf("post ret-1 code=%d body=%s", rr1.Code, rr1.Body.String())
	}

	// 2) Same TIN & phone in sup-2 must NOT collide with sup-1.
	rrOther := httptest.NewRecorder()
	r.ServeHTTP(rrOther, supplierAdminReq(http.MethodPost, "/v1/supplier/retailer-links",
		`{"retailer_id":"ret-other","tax_id_tin":"301234567","phone":"+998901112233","legal_name":"Baraka"}`, "sup-2"))
	if rrOther.Code != http.StatusOK {
		t.Fatalf("post ret-other code=%d body=%s", rrOther.Code, rrOther.Body.String())
	}
	var otherResp struct {
		Link           SupplierRetailerLink   `json:"link"`
		DuplicatePeers []SupplierRetailerLink `json:"duplicate_peers"`
	}
	if err := json.NewDecoder(rrOther.Body).Decode(&otherResp); err != nil {
		t.Fatalf("decode otherResp: %v", err)
	}
	if otherResp.Link.DedupStatus != entityresolution.DedupStatusClean || len(otherResp.DuplicatePeers) != 0 {
		t.Fatalf("cross-tenant link must remain CLEAN with 0 peers, got %+v", otherResp)
	}

	// 3) Duplicate phone under sup-1 triggers DUPLICATE_CANDIDATE cluster.
	rr2 := httptest.NewRecorder()
	r.ServeHTTP(rr2, supplierAdminReq(http.MethodPost, "/v1/supplier/retailer-links",
		`{"retailer_id":"ret-2","phone":"090-111-22-33","legal_name":"Baraka 2"}`, "sup-1"))
	if rr2.Code != http.StatusOK {
		t.Fatalf("post ret-2 code=%d body=%s", rr2.Code, rr2.Body.String())
	}
	var dupResp struct {
		Link           SupplierRetailerLink   `json:"link"`
		DuplicatePeers []SupplierRetailerLink `json:"duplicate_peers"`
	}
	if err := json.NewDecoder(rr2.Body).Decode(&dupResp); err != nil {
		t.Fatalf("decode dupResp: %v", err)
	}
	if dupResp.Link.DedupStatus != entityresolution.DedupStatusDuplicateCandidate || dupResp.Link.DedupClusterID == "" {
		t.Fatalf("expected DUPLICATE_CANDIDATE with cluster id, got %+v", dupResp.Link)
	}
	if len(dupResp.DuplicatePeers) != 1 || dupResp.DuplicatePeers[0].RetailerID != "ret-1" {
		t.Fatalf("expected ret-1 in duplicate_peers, got %+v", dupResp.DuplicatePeers)
	}

	// 4) Dedup endpoint shows 1 cluster in sup-1 and 0 in sup-2.
	rrDedup1 := httptest.NewRecorder()
	r.ServeHTTP(rrDedup1, supplierAdminReq(http.MethodGet, "/v1/supplier/retailer-links/dedup", "", "sup-1"))
	if rrDedup1.Code != http.StatusOK {
		t.Fatalf("dedup sup-1 code=%d body=%s", rrDedup1.Code, rrDedup1.Body.String())
	}
	var dedup1 struct {
		Candidates []SupplierRetailerLink `json:"candidates"`
		Clusters   []retailerDedupCluster `json:"clusters"`
	}
	if err := json.NewDecoder(rrDedup1.Body).Decode(&dedup1); err != nil {
		t.Fatalf("decode dedup1: %v", err)
	}
	if len(dedup1.Candidates) != 2 || len(dedup1.Clusters) != 1 {
		t.Fatalf("expected 2 candidates / 1 cluster for sup-1, got %+v", dedup1)
	}

	rrDedup2 := httptest.NewRecorder()
	r.ServeHTTP(rrDedup2, supplierAdminReq(http.MethodGet, "/v1/supplier/retailer-links/dedup", "", "sup-2"))
	var dedup2 struct {
		Candidates []SupplierRetailerLink `json:"candidates"`
	}
	if err := json.NewDecoder(rrDedup2.Body).Decode(&dedup2); err != nil {
		t.Fatalf("decode dedup2: %v", err)
	}
	if len(dedup2.Candidates) != 0 {
		t.Fatalf("sup-2 must see 0 dedup candidates, got %d", len(dedup2.Candidates))
	}

	// 5) Resolve ret-2 -> merge_into ret-1 cleanly marks ret-2 MERGED and auto-cleans ret-1.
	rrRes := httptest.NewRecorder()
	r.ServeHTTP(rrRes, supplierAdminReq(http.MethodPost, "/v1/supplier/retailer-links/ret-2/resolve",
		`{"action":"merge_into","merged_into_id":"ret-1"}`, "sup-1"))
	if rrRes.Code != http.StatusOK {
		t.Fatalf("resolve code=%d body=%s", rrRes.Code, rrRes.Body.String())
	}
	var resBody struct {
		Link         SupplierRetailerLink   `json:"link"`
		UpdatedPeers []SupplierRetailerLink `json:"updated_peers"`
	}
	if err := json.NewDecoder(rrRes.Body).Decode(&resBody); err != nil {
		t.Fatalf("decode resBody: %v", err)
	}
	if resBody.Link.DedupStatus != entityresolution.DedupStatusMerged || resBody.Link.MergedIntoID != "ret-1" {
		t.Fatalf("expected MERGED into ret-1, got %+v", resBody.Link)
	}
	if len(resBody.UpdatedPeers) != 1 || resBody.UpdatedPeers[0].DedupStatus != entityresolution.DedupStatusClean {
		t.Fatalf("expected surviving ret-1 auto-cleaned, got %+v", resBody.UpdatedPeers)
	}
}

func TestHandleExceptions_SurfacesExportRejectedAndTinDedup_AndResolves(t *testing.T) {
	now := time.Date(2026, 11, 13, 10, 0, 0, 0, time.UTC)
	svc := NewService(ServiceConfig{
		Repo:       &supplierOrdersTestRepo{},
		SupplierID: "sup-1",
		Now:        func() time.Time { return now },
	})

	onecRepo := partner.NewMemoryOnecPostingRepository()
	posting, _, err := onecRepo.CreateIfAbsent(context.Background(), partner.OnecPosting{
		PostingID:     "post-1",
		SupplierID:    "sup-1",
		DocumentKey:   "INV-2026-001",
		DocumentType:  partner.OnecDocJournalEntry,
		PayloadJSON:   `{"entry":"INV-2026-001"}`,
		ExposureMinor: 450000,
		Status:        partner.OnecStatusPending,
	})
	if err != nil {
		t.Fatalf("CreateIfAbsent: %v", err)
	}
	exp := int64(450000)
	if _, err := onecRepo.Reject(context.Background(), "sup-1", posting.PostingID, "account_62_mismatch", &exp, now); err != nil {
		t.Fatalf("Reject posting: %v", err)
	}
	svc.SetOnecPostings(onecRepo)

	if _, _, err := svc.RetailerLinks().Upsert(context.Background(), SupplierRetailerLink{
		SupplierID: "sup-1",
		RetailerID: "ret-a",
		TaxIDTIN:   "301234567",
		LegalName:  "Alpha LLC",
	}, now); err != nil {
		t.Fatalf("upsert ret-a: %v", err)
	}
	if _, _, err := svc.RetailerLinks().Upsert(context.Background(), SupplierRetailerLink{
		SupplierID: "sup-1",
		RetailerID: "ret-b",
		TaxIDTIN:   "301-234-567",
		LegalName:  "Alpha Copy",
	}, now); err != nil {
		t.Fatalf("upsert ret-b: %v", err)
	}

	rrEx := httptest.NewRecorder()
	svc.HandleExceptions(rrEx, supplierAdminReq(http.MethodGet, "/v1/supplier/exceptions", "", "sup-1"))
	if rrEx.Code != http.StatusOK {
		t.Fatalf("exceptions code=%d body=%s", rrEx.Code, rrEx.Body.String())
	}
	var exResp struct {
		Exceptions []SupplierExceptionRow `json:"exceptions"`
	}
	if err := json.NewDecoder(rrEx.Body).Decode(&exResp); err != nil {
		t.Fatalf("decode exceptions: %v", err)
	}

	var foundExport bool
	dedupCount := 0
	for _, row := range exResp.Exceptions {
		switch row.Kind {
		case "EXPORT_REJECTED":
			foundExport = true
			if row.MoneyAtRiskMinor != 450000 || row.PostingID != "post-1" || row.DocumentKey != "INV-2026-001" {
				t.Fatalf("unexpected EXPORT_REJECTED row: %+v", row)
			}
			if !strings.Contains(row.Note, "account_62_mismatch") {
				t.Fatalf("expected reject reason in Note, got %q", row.Note)
			}
		case "TIN_DEDUP_CANDIDATE":
			dedupCount++
			if row.DedupClusterID == "" || row.Severity == "" {
				t.Fatalf("expected cluster id and severity on TIN_DEDUP_CANDIDATE, got %+v", row)
			}
		}
	}
	if !foundExport || dedupCount != 2 {
		t.Fatalf("expected EXPORT_REJECTED + 2 TIN_DEDUP_CANDIDATE rows, got export=%v dedup=%d rows=%+v", foundExport, dedupCount, exResp.Exceptions)
	}

	// Resolve EXPORT_REJECTED (requeue -> PENDING) & TIN_DEDUP_CANDIDATE (merge_into -> CLEAN).
	r := chi.NewRouter()
	r.Post("/v1/supplier/exceptions/{kind}/{id}/resolve", HandleResolveException(ExceptionResolveDeps{
		OnecPostings:  onecRepo,
		RetailerLinks: svc.RetailerLinks(),
	}))

	rrRequeue := httptest.NewRecorder()
	r.ServeHTTP(rrRequeue, supplierAdminReq(http.MethodPost, "/v1/supplier/exceptions/EXPORT_REJECTED/post-1/resolve",
		`{"action":"requeue"}`, "sup-1"))
	if rrRequeue.Code != http.StatusOK {
		t.Fatalf("requeue code=%d body=%s", rrRequeue.Code, rrRequeue.Body.String())
	}
	gotPost, ok, err := onecRepo.Get(context.Background(), "sup-1", "post-1")
	if err != nil || !ok || gotPost.Status != partner.OnecStatusPending {
		t.Fatalf("expected requeued PENDING posting, got %+v ok=%v err=%v", gotPost, ok, err)
	}

	// Reject again then ack/close from Exception Desk.
	if _, err := onecRepo.Reject(context.Background(), "sup-1", "post-1", "tax_missing", &exp, now); err != nil {
		t.Fatalf("re-reject: %v", err)
	}
	rrAck := httptest.NewRecorder()
	r.ServeHTTP(rrAck, supplierAdminReq(http.MethodPost, "/v1/supplier/exceptions/EXPORT_REJECTED/post-1/resolve",
		`{"action":"ack","onec_ref":"1C-MANUAL-9"}`, "sup-1"))
	if rrAck.Code != http.StatusOK {
		t.Fatalf("ack code=%d body=%s", rrAck.Code, rrAck.Body.String())
	}
	gotAcked, _, _ := onecRepo.Get(context.Background(), "sup-1", "post-1")
	if gotAcked.Status != partner.OnecStatusAcked || gotAcked.OnecRef != "1C-MANUAL-9" {
		t.Fatalf("expected ACKED 1C-MANUAL-9, got %+v", gotAcked)
	}

	// Resolve TIN_DEDUP_CANDIDATE via merge_into.
	rrMerge := httptest.NewRecorder()
	r.ServeHTTP(rrMerge, supplierAdminReq(http.MethodPost, "/v1/supplier/exceptions/TIN_DEDUP_CANDIDATE/ret-b/resolve",
		`{"action":"merge_into","merged_into_id":"ret-a"}`, "sup-1"))
	if rrMerge.Code != http.StatusOK {
		t.Fatalf("merge code=%d body=%s", rrMerge.Code, rrMerge.Body.String())
	}
	retA, okA, _ := svc.RetailerLinks().Get(context.Background(), "sup-1", "ret-a")
	retB, okB, _ := svc.RetailerLinks().Get(context.Background(), "sup-1", "ret-b")
	if !okA || !okB || retB.DedupStatus != entityresolution.DedupStatusMerged || retA.DedupStatus != entityresolution.DedupStatusClean {
		t.Fatalf("unexpected resolved cluster state retA=%+v retB=%+v", retA, retB)
	}
}

func TestHandlePlanningSeasonalHolidaysSeed_UnavailableWithoutSpanner(t *testing.T) {
	svc := NewService(ServiceConfig{SupplierID: "sup-1"})
	rr := httptest.NewRecorder()
	svc.HandlePlanningSeasonalHolidaysSeed(rr, supplierAdminReq(http.MethodPost, "/v1/supplier/planning/seasonal-holidays/seed", "", "sup-1"))
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 without Spanner, got %d body=%s", rr.Code, rr.Body.String())
	}
}

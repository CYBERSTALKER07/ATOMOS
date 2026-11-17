package partner

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/catalog"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
)

type memCatalogRepo struct {
	products map[string]catalog.Product
}

func (m *memCatalogRepo) ListCategories(context.Context, string) ([]catalog.Category, error) {
	return nil, nil
}
func (m *memCatalogRepo) GetCategory(context.Context, string) (*catalog.Category, error) {
	return nil, nil
}
func (m *memCatalogRepo) CreateCategory(context.Context, catalog.Category, func(outbox.TxnBuffer) error) error {
	return nil
}
func (m *memCatalogRepo) ListProducts(_ context.Context, supplierID, _ string, _ bool) ([]catalog.Product, error) {
	var out []catalog.Product
	for _, p := range m.products {
		if p.SupplierID == supplierID {
			out = append(out, p)
		}
	}
	return out, nil
}
func (m *memCatalogRepo) ListDiscoverableProducts(context.Context, string, string, int64, int64) ([]catalog.Product, error) {
	return nil, nil
}
func (m *memCatalogRepo) GetProduct(_ context.Context, productID string) (*catalog.Product, error) {
	p, ok := m.products[productID]
	if !ok {
		return nil, errors.New("not found")
	}
	return &p, nil
}
func (m *memCatalogRepo) CreateProduct(_ context.Context, p catalog.Product, _ func(outbox.TxnBuffer) error) error {
	m.products[p.ProductID] = p
	return nil
}
func (m *memCatalogRepo) UpdateProduct(_ context.Context, p catalog.Product, _ func(outbox.TxnBuffer) error) error {
	m.products[p.ProductID] = p
	return nil
}
func (m *memCatalogRepo) ListCategorySuppliers(context.Context, string) ([]catalog.CategorySupplier, error) {
	return nil, nil
}
func (m *memCatalogRepo) SearchSuppliers(context.Context, string, int) ([]catalog.CategorySupplier, error) {
	return nil, nil
}

func TestMemoryOnecPostingRepository_Lifecycle(t *testing.T) {
	ctx := context.Background()
	repo := NewMemoryOnecPostingRepository()

	p1, created, err := repo.CreateIfAbsent(ctx, OnecPosting{
		SupplierID:    "sup-1",
		DocumentKey:   "1c:ar:e1",
		DocumentType:  OnecDocJournalEntry,
		ExportJobID:   "job-1",
		PayloadJSON:   `{"entry_id":"e1"}`,
		ExposureMinor: 125_000,
	})
	if err != nil || !created {
		t.Fatalf("first create: created=%v err=%v", created, err)
	}
	if p1.PostingID == "" || p1.Status != OnecStatusPending || p1.ExposureMinor != 125_000 {
		t.Fatalf("unexpected posting: %+v", p1)
	}

	// Idempotent duplicate on same (SupplierId, DocumentKey).
	pDup, createdDup, err := repo.CreateIfAbsent(ctx, OnecPosting{
		SupplierID:    "sup-1",
		DocumentKey:   "1c:ar:e1",
		DocumentType:  OnecDocJournalEntry,
		ExposureMinor: 999,
	})
	if err != nil || createdDup || pDup.PostingID != p1.PostingID {
		t.Fatalf("expected idempotent hit: pDup=%+v created=%v err=%v", pDup, createdDup, err)
	}

	// Same DocumentKey for a different supplier must be isolated.
	pOther, createdOther, err := repo.CreateIfAbsent(ctx, OnecPosting{
		SupplierID:    "sup-2",
		DocumentKey:   "1c:ar:e1",
		DocumentType:  OnecDocJournalEntry,
		ExposureMinor: 50_000,
	})
	if err != nil || !createdOther || pOther.PostingID == p1.PostingID {
		t.Fatalf("cross-supplier create failed: pOther=%+v created=%v err=%v", pOther, createdOther, err)
	}

	// Cross-tenant ack must fail with ErrOnecPostingNotFound.
	if _, err := repo.Ack(ctx, "sup-2", p1.PostingID, "REF-X", time.Now().UTC()); !errors.Is(err, ErrOnecPostingNotFound) {
		t.Fatalf("expected ErrOnecPostingNotFound cross-tenant, got %v", err)
	}

	// Reject updates reason, attempts, and exposure_minor.
	newExp := int64(130_000)
	rej, err := repo.Reject(ctx, "sup-1", p1.PostingID, "unknown SKU account", &newExp, time.Now().UTC())
	if err != nil {
		t.Fatalf("Reject: %v", err)
	}
	if rej.Status != OnecStatusRejected || rej.Attempts != 1 || rej.ExposureMinor != 130_000 || rej.RejectedAt == nil {
		t.Fatalf("unexpected rejected posting: %+v", rej)
	}

	rejectedList, err := repo.ListBySupplier(ctx, "sup-1", OnecStatusRejected, 50)
	if err != nil || len(rejectedList) != 1 {
		t.Fatalf("ListBySupplier(REJECTED) len=%d err=%v", len(rejectedList), err)
	}

	// Requeue resets to PENDING cleanly.
	reqd, err := repo.Requeue(ctx, "sup-1", p1.PostingID, time.Now().UTC())
	if err != nil || reqd.Status != OnecStatusPending || reqd.RejectReason != "" {
		t.Fatalf("Requeue: got %+v err=%v", reqd, err)
	}

	// Ack is atomic and idempotent.
	ack1, err := repo.Ack(ctx, "sup-1", p1.PostingID, "1C-DOC-001", time.Now().UTC())
	if err != nil || ack1.Status != OnecStatusAcked || ack1.OnecRef != "1C-DOC-001" || ack1.AckedAt == nil {
		t.Fatalf("Ack1: %+v err=%v", ack1, err)
	}
	ack2, err := repo.Ack(ctx, "sup-1", p1.DocumentKey, "1C-DOC-001", time.Now().UTC())
	if err != nil || ack2.Status != OnecStatusAcked || ack2.OnecRef != "1C-DOC-001" {
		t.Fatalf("Ack2 idempotent: %+v err=%v", ack2, err)
	}
}

func TestHandleOneCImport_DurableIdempotency(t *testing.T) {
	t.Setenv("DEFAULT_MARKET_CODE", "UZ")
	postings := NewMemoryOnecPostingRepository()
	catRepo := &memCatalogRepo{products: map[string]catalog.Product{}}
	catSvc := catalog.NewService(catRepo, nil, nil, nil, nil)

	// Failed upsert (catalog nil -> 422) must roll back idempotency marker so a fixed retry can succeed.
	svcBad := NewService(NewMemoryKeyRepository(), NewMemoryWebhookRepository(), nil, nil, nil)
	svcBad.SetOnecPostings(postings)
	hBad := &Handlers{Svc: svcBad}

	svc1 := NewService(NewMemoryKeyRepository(), NewMemoryWebhookRepository(), nil, catSvc, nil)
	svc1.SetOnecPostings(postings)
	h1 := &Handlers{Svc: svc1}

	// Second independent Handlers/Service instance sharing durable OnecPostingRepository (simulates restart).
	svc2 := NewService(NewMemoryKeyRepository(), NewMemoryWebhookRepository(), nil, catSvc, nil)
	svc2.SetOnecPostings(postings)
	h2 := &Handlers{Svc: svc2}

	payload := []byte(`{"external_doc_id":"DOC-1C-42","products":[{"external_id":"sku-1","name":"Black Tea 100g","price_minor":12000}]}`)
	mkReq := func() *http.Request {
		r := httptest.NewRequest(http.MethodPost, "/partner/v1/adapters/onec/import", bytes.NewReader(payload))
		return r.WithContext(auth.WithClaims(WithPrincipal(r.Context(), Principal{
			TenantType: TenantSupplier, TenantID: "sup-1c", Scopes: []string{ScopeCatalogWrite},
		}), auth.Claims{MarketCode: "UZ", SupplierID: "sup-1c"}))
	}

	rrBad := httptest.NewRecorder()
	hBad.HandleOneCImport(rrBad, mkReq())
	if rrBad.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422 when catalog unavailable, got %d body=%s", rrBad.Code, rrBad.Body.String())
	}

	rr1 := httptest.NewRecorder()
	h1.HandleOneCImport(rr1, mkReq())
	if rr1.Code != http.StatusOK || !strings.Contains(rr1.Body.String(), `"accepted"`) {
		t.Fatalf("first import: status=%d body=%s", rr1.Code, rr1.Body.String())
	}

	rr2 := httptest.NewRecorder()
	h2.HandleOneCImport(rr2, mkReq())
	if rr2.Code != http.StatusOK || !strings.Contains(rr2.Body.String(), `"duplicate"`) {
		t.Fatalf("second import across instance must return duplicate: status=%d body=%s", rr2.Code, rr2.Body.String())
	}
}

func TestHandleOneCPendingAckReject_PartnerAndSupplierRoutes(t *testing.T) {
	keys := NewMemoryKeyRepository()
	svc := NewService(keys, NewMemoryWebhookRepository(), nil, nil, nil)
	postings := NewMemoryOnecPostingRepository()
	svc.SetOnecPostings(postings)

	ctx := context.Background()
	issued, err := svc.IssueKey(ctx, TenantSupplier, "sup-route", "admin", []string{ScopeExportsRead})
	if err != nil {
		t.Fatal(err)
	}
	p1, _, err := postings.CreateIfAbsent(ctx, OnecPosting{
		SupplierID: "sup-route", DocumentKey: "1c:ar:inv1", PayloadJSON: `{"amount_minor":50000}`, ExposureMinor: 50_000,
	})
	if err != nil {
		t.Fatal(err)
	}
	p2, _, err := postings.CreateIfAbsent(ctx, OnecPosting{
		SupplierID: "sup-route", DocumentKey: "1c:ar:inv2", PayloadJSON: `{"amount_minor":75000}`, ExposureMinor: 75_000,
	})
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = postings.CreateIfAbsent(ctx, OnecPosting{
		SupplierID: "sup-other", DocumentKey: "1c:ar:other", PayloadJSON: `{}`, ExposureMinor: 10_000,
	})
	if err != nil {
		t.Fatal(err)
	}

	r := chi.NewRouter()
	h := &Handlers{Svc: svc}
	RegisterPartnerRoutes(r, keys, h)
	// Mount admin/supplier routes with test JWT claims injection middleware.
	r.Group(func(sr chi.Router) {
		sr.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{
					Subject:    "admin-1",
					Role:       auth.RoleAdmin,
					SupplierID: "sup-route",
				}))
				next.ServeHTTP(w, req)
			})
		})
		RegisterAdminKeyRoutes(sr, h)
	})

	// 1. Partner key GET /partner/v1/adapters/onec/pending returns only sup-route's 2 PENDING postings.
	req := httptest.NewRequest(http.MethodGet, "/partner/v1/adapters/onec/pending", nil)
	req.Header.Set("Authorization", "Bearer "+issued.Secret)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("partner pending status=%d body=%s", rr.Code, rr.Body.String())
	}
	var listResp struct {
		Postings []OnecPosting `json:"postings"`
		Count    int           `json:"count"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&listResp); err != nil {
		t.Fatal(err)
	}
	if listResp.Count != 2 || len(listResp.Postings) != 2 {
		t.Fatalf("expected 2 pending postings, got %+v", listResp)
	}

	// 2. Partner key POST /partner/v1/adapters/onec/ack
	ackReq := httptest.NewRequest(http.MethodPost, "/partner/v1/adapters/onec/ack",
		bytes.NewReader([]byte(`{"posting_id":"`+p1.PostingID+`","onec_ref":"1C-2026-88"}`)))
	ackReq.Header.Set("Authorization", "Bearer "+issued.Secret)
	ackRR := httptest.NewRecorder()
	r.ServeHTTP(ackRR, ackReq)
	if ackRR.Code != http.StatusOK || !strings.Contains(ackRR.Body.String(), `"1C-2026-88"`) {
		t.Fatalf("partner ack status=%d body=%s", ackRR.Code, ackRR.Body.String())
	}

	// 3. Supplier JWT POST /v1/supplier/partner-onec/reject
	rejReq := httptest.NewRequest(http.MethodPost, "/v1/supplier/partner-onec/reject",
		bytes.NewReader([]byte(`{"document_key":"`+p2.DocumentKey+`","reject_reason":"VAT period closed","exposure_minor":80000}`)))
	rejRR := httptest.NewRecorder()
	r.ServeHTTP(rejRR, rejReq)
	if rejRR.Code != http.StatusOK || !strings.Contains(rejRR.Body.String(), `"VAT period closed"`) {
		t.Fatalf("supplier reject status=%d body=%s", rejRR.Code, rejRR.Body.String())
	}

	// 4. Supplier JWT GET /v1/supplier/partner-onec/pending?status=REJECTED
	supListReq := httptest.NewRequest(http.MethodGet, "/v1/supplier/partner-onec/pending?status=REJECTED", nil)
	supListRR := httptest.NewRecorder()
	r.ServeHTTP(supListRR, supListReq)
	if supListRR.Code != http.StatusOK {
		t.Fatalf("supplier rejected list status=%d body=%s", supListRR.Code, supListRR.Body.String())
	}
	var rejList struct {
		Postings []OnecPosting `json:"postings"`
	}
	if err := json.NewDecoder(supListRR.Body).Decode(&rejList); err != nil {
		t.Fatal(err)
	}
	if len(rejList.Postings) != 1 || rejList.Postings[0].ExposureMinor != 80_000 {
		t.Fatalf("expected 1 rejected posting with exposure 80000, got %+v", rejList.Postings)
	}
}

func TestExportWorker_RecordOnecJournalPostingsIdempotent(t *testing.T) {
	ctx := context.Background()
	exports := NewMemoryExportRepository()
	sftp := NewMemorySftpConfigRepository()
	w := NewExportWorker(exports, sftp, nil, nil)
	repo := w.OnecPostings()
	if repo == nil {
		t.Fatal("expected default MemoryOnecPostingRepository on worker")
	}

	rows := []map[string]any{
		{"source": "ar", "entry_id": "e1", "entry_type": "OPEN", "amount_minor": int64(112_000)},
		{"source": "payment", "entry_id": "p1", "entry_type": "SESSION_CAPTURED", "amount_minor": int64(50_000)},
	}
	job := ExportJob{
		JobID:      "job-1c-1",
		TenantType: TenantSupplier,
		TenantID:   "sup-w",
		Resource:   ExportResourceJournals,
		Format:     ExportFormatXML,
	}
	if err := w.recordOnecJournalPostings(ctx, job, rows); err != nil {
		t.Fatalf("first record: %v", err)
	}
	// Re-run with overlapping job -> must remain idempotent (2 postings, not 4).
	job2 := job
	job2.JobID = "job-1c-2"
	if err := w.recordOnecJournalPostings(ctx, job2, rows); err != nil {
		t.Fatalf("second record: %v", err)
	}
	got, err := repo.ListBySupplier(ctx, "sup-w", OnecStatusPending, 50)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 idempotent postings, got %d", len(got))
	}
}

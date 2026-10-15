package catalog

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/idempotency"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
)

type mockCatalogRepo struct {
	mu         sync.Mutex
	categories map[string]Category
	products   map[string]Product
	events     []outbox.Event
}

func newMockCatalogRepo() *mockCatalogRepo {
	return &mockCatalogRepo{
		categories: make(map[string]Category),
		products:   make(map[string]Product),
	}
}

type testTxnBuffer struct {
	repo *mockCatalogRepo
}

func (b *testTxnBuffer) BufferOutbox(_ context.Context, e outbox.Event) error {
	b.repo.events = append(b.repo.events, e)
	return nil
}

func (m *mockCatalogRepo) ListCategories(_ context.Context, supplierID string) ([]Category, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []Category
	for _, c := range m.categories {
		if c.SupplierID == supplierID {
			out = append(out, c)
		}
	}
	return out, nil
}

func (m *mockCatalogRepo) GetCategory(_ context.Context, categoryID string) (*Category, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	c, ok := m.categories[categoryID]
	if !ok {
		return nil, errors.New("category not found")
	}
	return &c, nil
}

func (m *mockCatalogRepo) CreateCategory(_ context.Context, cat Category, emit func(outbox.TxnBuffer) error) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if emit != nil {
		buf := &testTxnBuffer{repo: m}
		if err := emit(buf); err != nil {
			return err
		}
	}
	m.categories[cat.CategoryID] = cat
	return nil
}

func (m *mockCatalogRepo) ListProducts(_ context.Context, supplierID, categoryID string, activeOnly bool) ([]Product, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []Product
	for _, p := range m.products {
		if p.SupplierID == supplierID {
			if categoryID != "" && p.CategoryID != categoryID {
				continue
			}
			if activeOnly && !p.IsActive {
				continue
			}
			out = append(out, p)
		}
	}
	return out, nil
}

func (m *mockCatalogRepo) ListDiscoverableProducts(_ context.Context, categoryID, marketCode string, limit, offset int64) ([]Product, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []Product
	for _, p := range m.products {
		if p.IsActive {
			out = append(out, p)
		}
	}
	return out, nil
}

func (m *mockCatalogRepo) GetProduct(_ context.Context, productID string) (*Product, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	p, ok := m.products[productID]
	if !ok {
		return nil, errors.New("product not found")
	}
	return &p, nil
}

func (m *mockCatalogRepo) CreateProduct(_ context.Context, p Product, emit func(outbox.TxnBuffer) error) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if emit != nil {
		buf := &testTxnBuffer{repo: m}
		if err := emit(buf); err != nil {
			return err
		}
	}
	p.Version = 1
	m.products[p.ProductID] = p
	return nil
}

func (m *mockCatalogRepo) UpdateProduct(_ context.Context, p Product, emit func(outbox.TxnBuffer) error) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	curr, ok := m.products[p.ProductID]
	if !ok {
		return errors.New("product not found")
	}
	if curr.Version != p.Version {
		return ErrOptimisticLockConflict
	}
	if emit != nil {
		buf := &testTxnBuffer{repo: m}
		if err := emit(buf); err != nil {
			return err
		}
	}
	p.Version = curr.Version + 1
	m.products[p.ProductID] = p
	return nil
}

func (m *mockCatalogRepo) ListCategorySuppliers(_ context.Context, categoryID string) ([]CategorySupplier, error) {
	return nil, nil
}

func (m *mockCatalogRepo) SearchSuppliers(_ context.Context, query string, limit int) ([]CategorySupplier, error) {
	return nil, nil
}

func supplierAuthContext(supplierID string) context.Context {
	claims := auth.Claims{
		Subject:    "usr-1",
		SupplierID: supplierID,
		Role:       auth.RoleAdmin,
	}
	return auth.WithClaims(context.Background(), claims)
}

func TestCatalog_CreateProduct_StrictIdempotency(t *testing.T) {
	repo := newMockCatalogRepo()
	idem := idempotency.NewInMemoryStore()
	svc := NewService(repo, nil, nil, nil, nil)
	svc.SetIdempotencyStore(idem)

	reqBody := []byte(`{
		"supplier_id": "sup-1",
		"category_id": "cat-1",
		"name": "Organic Milk",
		"price_minor": 15000,
		"currency": "UZS",
		"stock_quantity": 100,
		"unit": "UNIT",
		"handling_class": "COLD_CHAIN",
		"storage_temp_min_c": 2.0,
		"storage_temp_max_c": 6.0
	}`)

	// 1. Missing Idempotency-Key -> 400 Bad Request
	req := httptest.NewRequest(http.MethodPost, "/v1/catalog/products", bytes.NewReader(reqBody)).WithContext(supplierAuthContext("sup-1"))
	rec := httptest.NewRecorder()
	svc.HandleCreateProduct(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 without idempotency key, got %d: %s", rec.Code, rec.Body.String())
	}

	// 2. With Idempotency-Key -> 201 Created and outbox events emitted
	req = httptest.NewRequest(http.MethodPost, "/v1/catalog/products", bytes.NewReader(reqBody)).WithContext(supplierAuthContext("sup-1"))
	req.Header.Set("Idempotency-Key", "create-prod-1001")
	rec = httptest.NewRecorder()
	svc.HandleCreateProduct(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 created, got %d: %s", rec.Code, rec.Body.String())
	}

	var created Product
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode created product: %v", err)
	}
	if created.Name != "Organic Milk" || created.PriceMinor != 15000 {
		t.Fatalf("unexpected product state: %+v", created)
	}

	// Verify Outbox Events
	repo.mu.Lock()
	var foundCreated, foundHandling bool
	for _, e := range repo.events {
		var base events.BaseEvent
		_ = json.Unmarshal(e.Payload, &base)
		if base.Type == events.EventProductCreated {
			foundCreated = true
			var pe events.ProductEvent
			_ = json.Unmarshal(e.Payload, &pe)
			if pe.ProductID != created.ProductID || pe.PriceMinor != 15000 || pe.HandlingClass != "COLD_CHAIN" {
				t.Fatalf("unexpected product created event: %+v", pe)
			}
		}
		if base.Type == events.EventProductHandlingUpdated {
			foundHandling = true
		}
	}
	repo.mu.Unlock()

	if !foundCreated {
		t.Fatalf("expected PRODUCT_CREATED outbox event to be buffered")
	}
	if !foundHandling {
		t.Fatalf("expected PRODUCT_HANDLING_UPDATED outbox event to be buffered")
	}

	// 3. Replay with identical key -> returns cached 201
	reqReplay := httptest.NewRequest(http.MethodPost, "/v1/catalog/products", bytes.NewReader(reqBody)).WithContext(supplierAuthContext("sup-1"))
	reqReplay.Header.Set("Idempotency-Key", "create-prod-1001")
	recReplay := httptest.NewRecorder()
	svc.HandleCreateProduct(recReplay, reqReplay)
	if recReplay.Code != http.StatusCreated {
		t.Fatalf("expected replayed 201, got %d", recReplay.Code)
	}

	// 4. Same key with different payload -> 409 Conflict
	differentBody := []byte(`{
		"supplier_id": "sup-1",
		"category_id": "cat-1",
		"name": "Different Milk",
		"price_minor": 18000,
		"currency": "UZS"
	}`)
	reqConflict := httptest.NewRequest(http.MethodPost, "/v1/catalog/products", bytes.NewReader(differentBody)).WithContext(supplierAuthContext("sup-1"))
	reqConflict.Header.Set("Idempotency-Key", "create-prod-1001")
	recConflict := httptest.NewRecorder()
	svc.HandleCreateProduct(recConflict, reqConflict)
	if recConflict.Code != http.StatusConflict {
		t.Fatalf("expected 409 on idempotency conflict, got %d", recConflict.Code)
	}
}

func TestCatalog_UpdateProduct_OptimisticLockingAndEvents(t *testing.T) {
	repo := newMockCatalogRepo()
	idem := idempotency.NewInMemoryStore()
	svc := NewService(repo, nil, nil, nil, nil)
	svc.SetIdempotencyStore(idem)

	initialProduct := Product{
		ProductID:     "prod-test-1",
		SupplierID:    "sup-1",
		CategoryID:    "cat-1",
		Name:          "Original Juice",
		PriceMinor:    10000,
		Currency:      "UZS",
		StockQuantity: 50,
		IsActive:      true,
		Version:       1,
	}
	repo.products[initialProduct.ProductID] = initialProduct

	// 1. Missing Idempotency-Key -> 400
	updateBody := []byte(`{
		"name": "Updated Juice",
		"price_minor": 12000,
		"version": 1
	}`)
	req := httptest.NewRequest(http.MethodPut, "/v1/catalog/products/prod-test-1", bytes.NewReader(updateBody)).WithContext(supplierAuthContext("sup-1"))
	req.SetPathValue("productID", "prod-test-1")
	rec := httptest.NewRecorder()
	svc.HandleUpdateProduct(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 without idempotency key, got %d", rec.Code)
	}

	// 2. Version Conflict: Sending stale version 99 -> 409 Conflict
	conflictBody := []byte(`{
		"name": "Updated Juice",
		"price_minor": 12000,
		"version": 99
	}`)
	req = httptest.NewRequest(http.MethodPut, "/v1/catalog/products/prod-test-1", bytes.NewReader(conflictBody)).WithContext(supplierAuthContext("sup-1"))
	req.SetPathValue("productID", "prod-test-1")
	req.Header.Set("Idempotency-Key", "upd-key-conflict")
	rec = httptest.NewRecorder()
	svc.HandleUpdateProduct(rec, req)
	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409 version conflict, got %d: %s", rec.Code, rec.Body.String())
	}

	// 3. Successful update with price change -> emits PRODUCT_PRICE_CHANGED
	repo.events = nil
	req = httptest.NewRequest(http.MethodPut, "/v1/catalog/products/prod-test-1", bytes.NewReader(updateBody)).WithContext(supplierAuthContext("sup-1"))
	req.SetPathValue("productID", "prod-test-1")
	req.Header.Set("Idempotency-Key", "upd-key-success")
	rec = httptest.NewRecorder()
	svc.HandleUpdateProduct(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 ok, got %d: %s", rec.Code, rec.Body.String())
	}

	repo.mu.Lock()
	var foundPriceChanged bool
	for _, e := range repo.events {
		var pe events.ProductEvent
		_ = json.Unmarshal(e.Payload, &pe)
		if pe.Type == events.EventProductPriceChanged {
			foundPriceChanged = true
			if pe.PreviousPriceMinor != 10000 || pe.PriceMinor != 12000 {
				t.Fatalf("expected price 10000 -> 12000, got prev %d curr %d", pe.PreviousPriceMinor, pe.PriceMinor)
			}
		}
	}
	repo.mu.Unlock()

	if !foundPriceChanged {
		t.Fatalf("expected PRODUCT_PRICE_CHANGED outbox event")
	}

	// 4. Deactivate product -> emits PRODUCT_DEACTIVATED
	repo.events = nil
	deactBody := []byte(`{
		"is_active": false,
		"version": 2
	}`)
	req = httptest.NewRequest(http.MethodPut, "/v1/catalog/products/prod-test-1", bytes.NewReader(deactBody)).WithContext(supplierAuthContext("sup-1"))
	req.SetPathValue("productID", "prod-test-1")
	req.Header.Set("Idempotency-Key", "upd-key-deact")
	rec = httptest.NewRecorder()
	svc.HandleUpdateProduct(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 ok on deactivation, got %d: %s", rec.Code, rec.Body.String())
	}

	repo.mu.Lock()
	var foundDeactivated bool
	for _, e := range repo.events {
		var pe events.ProductEvent
		_ = json.Unmarshal(e.Payload, &pe)
		if pe.Type == events.EventProductDeactivated {
			foundDeactivated = true
			if pe.IsActive {
				t.Fatalf("expected is_active=false in event")
			}
		}
	}
	repo.mu.Unlock()

	if !foundDeactivated {
		t.Fatalf("expected PRODUCT_DEACTIVATED outbox event")
	}
}

func TestCatalog_CreateCategory_StrictIdempotencyAndEvents(t *testing.T) {
	repo := newMockCatalogRepo()
	idem := idempotency.NewInMemoryStore()
	svc := NewService(repo, nil, nil, nil, nil)
	svc.SetIdempotencyStore(idem)

	body := []byte(`{
		"supplier_id": "sup-1",
		"name": "Dairy & Beverages",
		"icon_key": "beverages_icon",
		"sort_order": 5
	}`)

	// Missing key -> 400
	req := httptest.NewRequest(http.MethodPost, "/v1/catalog/categories", bytes.NewReader(body)).WithContext(supplierAuthContext("sup-1"))
	rec := httptest.NewRecorder()
	svc.HandleCreateCategory(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 without key, got %d", rec.Code)
	}

	// With key -> 201 Created and CATEGORY_CREATED outbox event
	req = httptest.NewRequest(http.MethodPost, "/v1/catalog/categories", bytes.NewReader(body)).WithContext(supplierAuthContext("sup-1"))
	req.Header.Set("Idempotency-Key", "cat-key-1")
	rec = httptest.NewRecorder()
	svc.HandleCreateCategory(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}

	repo.mu.Lock()
	var foundCategoryCreated bool
	for _, e := range repo.events {
		var ce events.CategoryEvent
		_ = json.Unmarshal(e.Payload, &ce)
		if ce.Type == events.EventCategoryCreated {
			foundCategoryCreated = true
			if ce.Name != "Dairy & Beverages" || ce.SupplierID != "sup-1" {
				t.Fatalf("unexpected category event: %+v", ce)
			}
		}
	}
	repo.mu.Unlock()

	if !foundCategoryCreated {
		t.Fatalf("expected CATEGORY_CREATED outbox event")
	}
}

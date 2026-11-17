package storageroutes

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/storage"
)

type fakeVaultStore struct {
	dossiers map[string]*storage.EvidenceDossier
	items    map[string][]*storage.EvidenceItem
}

func newFakeVaultStore() *fakeVaultStore {
	return &fakeVaultStore{
		dossiers: map[string]*storage.EvidenceDossier{},
		items:    map[string][]*storage.EvidenceItem{},
	}
}

func (f *fakeVaultStore) CreateDossier(_ context.Context, targetID, targetType string) (*storage.EvidenceDossier, error) {
	d := &storage.EvidenceDossier{
		DossierID:  "dos-1",
		TargetID:   targetID,
		TargetType: targetType,
		Status:     storage.DossierStatusOpen,
	}
	f.dossiers[d.DossierID] = d
	return d, nil
}

func (f *fakeVaultStore) GetDossier(_ context.Context, dossierID string) (*storage.EvidenceDossier, []*storage.EvidenceItem, error) {
	d, ok := f.dossiers[dossierID]
	if !ok {
		return nil, nil, storage.ErrDossierNotFound
	}
	return d, f.items[dossierID], nil
}

func (f *fakeVaultStore) AddItem(_ context.Context, dossierID string, req *storage.EvidenceItem) (*storage.EvidenceItem, error) {
	d, ok := f.dossiers[dossierID]
	if !ok {
		return nil, storage.ErrDossierNotFound
	}
	if d.Status == storage.DossierStatusSealed {
		return nil, storage.ErrDossierSealed
	}
	cp := *req
	cp.DossierID = dossierID
	if cp.ItemID == "" {
		cp.ItemID = "item-1"
	}
	f.items[dossierID] = append(f.items[dossierID], &cp)
	return &cp, nil
}

func (f *fakeVaultStore) SealDossier(_ context.Context, dossierID string) error {
	d, ok := f.dossiers[dossierID]
	if !ok {
		return storage.ErrDossierNotFound
	}
	if d.Status == storage.DossierStatusSealed {
		return storage.ErrDossierSealed
	}
	d.Status = storage.DossierStatusSealed
	return nil
}

func TestAddItem_AuthAndValidationRunBeforeTicketGeneration(t *testing.T) {
	// Keep REQUIRE_INFRA_ADAPTERS=true with nil Client so reaching GenerateUploadTicketFor returns 503.
	// Any unauthenticated or invalid request MUST fail with 401/403/400 BEFORE reaching ticket generation.
	t.Setenv("REQUIRE_INFRA_ADAPTERS", "true")
	prevClient := storage.Client
	storage.Client = nil
	t.Cleanup(func() { storage.Client = prevClient })

	vault := newFakeVaultStore()
	vault.dossiers["dos-1"] = &storage.EvidenceDossier{DossierID: "dos-1", Status: storage.DossierStatusOpen}

	validBody, _ := json.Marshal(AddItemRequest{
		ItemType:  "IMAGE",
		FileHash:  "sha256:abc",
		MimeType:  "image/jpeg",
		SizeBytes: 1024,
		Extension: "jpg",
	})

	mountWithClaims := func(claims *auth.Claims) http.Handler {
		r := chi.NewRouter()
		if claims != nil {
			r.Use(func(next http.Handler) http.Handler {
				return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
					next.ServeHTTP(w, req.WithContext(auth.WithClaims(req.Context(), *claims)))
				})
			})
		}
		MountWithStore(r, vault)
		return r
	}

	// 1. Unauthenticated must return 401 before touching storage (not 503).
	req := httptest.NewRequest(http.MethodPost, "/dossiers/dos-1/items", bytes.NewReader(validBody))
	rec := httptest.NewRecorder()
	mountWithClaims(nil).ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated: want 401, got %d (%s)", rec.Code, rec.Body.String())
	}

	// 2. Disallowed role (FACTORY) must return 403.
	req = httptest.NewRequest(http.MethodPost, "/dossiers/dos-1/items", bytes.NewReader(validBody))
	rec = httptest.NewRecorder()
	mountWithClaims(&auth.Claims{Subject: "u1", Role: auth.RoleFactory}).ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("wrong role: want 403, got %d (%s)", rec.Code, rec.Body.String())
	}

	// 3. Disallowed extension (.svg) must return 400 before storage (not 503).
	badExtBody, _ := json.Marshal(AddItemRequest{
		ItemType:  "IMAGE",
		FileHash:  "sha256:abc",
		SizeBytes: 1024,
		Extension: "svg",
	})
	retailerClaims := &auth.Claims{
		Subject:        "ret-user-9",
		RetailerOrgID:  "ret-org-42",
		RetailerUserID: "ret-user-9",
		Role:           auth.RoleRetailer,
	}
	req = httptest.NewRequest(http.MethodPost, "/dossiers/dos-1/items", bytes.NewReader(badExtBody))
	rec = httptest.NewRecorder()
	mountWithClaims(retailerClaims).ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("bad extension: want 400, got %d (%s)", rec.Code, rec.Body.String())
	}

	// 4. Out-of-bounds size (0 or >10 MiB) must return 400 before storage (not 503).
	for _, badSize := range []int64{0, -10, storage.MaxEvidenceUploadBytes + 1} {
		badSizeBody, _ := json.Marshal(AddItemRequest{
			ItemType:  "IMAGE",
			FileHash:  "sha256:abc",
			MimeType:  "image/png",
			SizeBytes: badSize,
			Extension: "png",
		})
		req = httptest.NewRequest(http.MethodPost, "/dossiers/dos-1/items", bytes.NewReader(badSizeBody))
		rec = httptest.NewRecorder()
		mountWithClaims(retailerClaims).ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("size %d: want 400, got %d (%s)", badSize, rec.Code, rec.Body.String())
		}
	}

	// 5. Valid retailer request reaches storage ticket generation (503 since Client=nil + fail-closed).
	req = httptest.NewRequest(http.MethodPost, "/dossiers/dos-1/items", bytes.NewReader(validBody))
	rec = httptest.NewRecorder()
	mountWithClaims(retailerClaims).ServeHTTP(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("valid fail-closed no-client request: want 503, got %d (%s)", rec.Code, rec.Body.String())
	}
}

func TestAddItem_RetailerScopedClaimsPathAndPlaceholderSuccess(t *testing.T) {
	t.Setenv("REQUIRE_INFRA_ADAPTERS", "false")
	t.Setenv("PEGASUSX_ENV", "local")
	prevClient := storage.Client
	storage.Client = nil
	t.Cleanup(func() { storage.Client = prevClient })

	vault := newFakeVaultStore()
	vault.dossiers["dos-99"] = &storage.EvidenceDossier{DossierID: "dos-99", Status: storage.DossierStatusOpen}

	retailerClaims := auth.Claims{
		Subject:        "owner-sub",
		RetailerOrgID:  "ret-org-77",
		RetailerUserID: "user-77",
		Role:           auth.RoleRetailer,
	}

	scope, uploader := resolveDossierActor(retailerClaims)
	if scope != "ret-org-77" || uploader != "user-77" {
		t.Fatalf("unexpected actor resolution: scope=%q uploader=%q", scope, uploader)
	}
	legacyClaims := auth.Claims{Subject: "legacy-ret", Role: auth.RoleRetailer}
	legacyScope, legacyUploader := resolveDossierActor(legacyClaims)
	if legacyScope != "legacy-ret" || legacyUploader != "legacy-ret" {
		t.Fatalf("unexpected legacy actor resolution: scope=%q uploader=%q", legacyScope, legacyUploader)
	}

	prefix := "claims/" + scope + "/dos-99"
	if !strings.HasPrefix(prefix, "claims/ret-org-77/dos-99") {
		t.Fatalf("unexpected claims prefix %q", prefix)
	}

	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			next.ServeHTTP(w, req.WithContext(auth.WithClaims(req.Context(), retailerClaims)))
		})
	})
	MountWithStore(r, vault)

	body, _ := json.Marshal(AddItemRequest{
		ItemType:  "IMAGE",
		FileHash:  "sha256:feed",
		SizeBytes: storage.MaxEvidenceUploadBytes,
		Extension: "webp",
	})
	req := httptest.NewRequest(http.MethodPost, "/dossiers/dos-99/items", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("retailer add item: want 201, got %d (%s)", rec.Code, rec.Body.String())
	}
	items := vault.items["dos-99"]
	if len(items) != 1 || items[0].UploaderUserID != "user-77" || items[0].MimeType != "image/webp" {
		t.Fatalf("unexpected stored item: %+v", items)
	}
}

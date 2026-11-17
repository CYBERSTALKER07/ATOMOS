package auth

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestTimeBoxedAuditorJWT_ParseAndRequireAuditorActive(t *testing.T) {
	t.Parallel()
	const secret = "test-auditor-timebox-secret-2026-12345"
	kr := NewKeyring(secret)

	future := time.Now().UTC().Add(2 * time.Hour)
	validTok, err := Issue(Claims{
		Subject:          "ext-aud-1",
		Role:             RoleAdmin,
		SupplierID:       "sup-1",
		SupplierRole:     Role(SupplierRoleAuditor),
		ActorType:        ActorTypeHuman,
		SupplierPerms:    []string{PermAuditRead},
		AuditorExpiresAt: &future,
	}, IssueOptions{Secret: secret})
	if err != nil {
		t.Fatalf("Issue valid auditor token: %v", err)
	}

	parsed, err := ParseTokenWithKeyring(validTok, kr)
	if err != nil {
		t.Fatalf("ParseTokenWithKeyring(valid): %v", err)
	}
	if parsed.AuditorExpiresAt == nil || parsed.AuditorExpiresAt.Unix() != future.Unix() {
		t.Fatalf("AuditorExpiresAt = %v, want %v", parsed.AuditorExpiresAt, future)
	}
	if !IsReadOnlyAuditor(parsed) || !HasSupplierPerm(parsed, PermAuditRead) || HasSupplierPerm(parsed, PermJournalPost) {
		t.Fatalf("unexpected auditor perms: %+v", parsed)
	}

	// Expired auditor token must fail ParseTokenWithKeyring & ParseWithKeyring with ErrAuditorAccessExpired.
	past := time.Now().UTC().Add(-2 * time.Minute)
	expiredTok, err := Issue(Claims{
		Subject:          "ext-aud-2",
		Role:             RoleAdmin,
		SupplierID:       "sup-1",
		SupplierRole:     Role(SupplierRoleAuditor),
		ActorType:        ActorTypeHuman,
		SupplierPerms:    []string{PermAuditRead},
		AuditorExpiresAt: &past,
	}, IssueOptions{Secret: secret, TTL: time.Hour})
	if err != nil {
		t.Fatalf("Issue expired auditor token: %v", err)
	}
	if _, err := ParseTokenWithKeyring(expiredTok, kr); !errors.Is(err, ErrAuditorAccessExpired) {
		t.Fatalf("ParseTokenWithKeyring(expired) err = %v, want ErrAuditorAccessExpired", err)
	}
	if _, err := ParseIgnoreExpiry(expiredTok, secret); !errors.Is(err, ErrAuditorAccessExpired) {
		t.Fatalf("ParseIgnoreExpiry(expired) err = %v, want ErrAuditorAccessExpired", err)
	}

	// HTTP middleware chain: SessionAuth + RequireAuditorActive + RequireSupplierPerm(PermAuditRead).
	handler := SessionAuthWithKeyring(kr)(RequireAuditorActive(RequireSupplierPerm(PermAuditRead)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))))

	// Active token -> 200 OK.
	reqOK := httptest.NewRequest(http.MethodGet, "/v1/finance/packs", nil)
	reqOK.Header.Set("Authorization", "Bearer "+validTok)
	recOK := httptest.NewRecorder()
	handler.ServeHTTP(recOK, reqOK)
	if recOK.Code != http.StatusOK {
		t.Fatalf("active auditor GET = %d body=%s, want 200", recOK.Code, recOK.Body.String())
	}

	// Expired token -> 401 auditor_access_expired.
	reqExp := httptest.NewRequest(http.MethodGet, "/v1/finance/packs", nil)
	reqExp.Header.Set("Authorization", "Bearer "+expiredTok)
	recExp := httptest.NewRecorder()
	handler.ServeHTTP(recExp, reqExp)
	if recExp.Code != http.StatusUnauthorized {
		t.Fatalf("expired auditor GET = %d body=%s, want 401", recExp.Code, recExp.Body.String())
	}
	var errBody map[string]string
	if err := json.Unmarshal(recExp.Body.Bytes(), &errBody); err != nil {
		t.Fatalf("decode 401 json: %v", err)
	}
	if errBody["error"] != "auditor_access_expired" || errBody["code"] != "auditor_access_expired" {
		t.Fatalf("unexpected 401 payload: %+v", errBody)
	}

	// Direct claims injection with past AuditorExpiresAt also returns 401 auditor_access_expired.
	directMW := RequireAuditorActive(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	reqDirect := httptest.NewRequest(http.MethodGet, "/v1/finance/packs", nil)
	reqDirect = reqDirect.WithContext(WithClaims(reqDirect.Context(), Claims{
		Subject:          "ext-aud-3",
		Role:             RoleAdmin,
		SupplierID:       "sup-1",
		SupplierRole:     Role(SupplierRoleAuditor),
		AuditorExpiresAt: &past,
	}))
	recDirect := httptest.NewRecorder()
	directMW.ServeHTTP(recDirect, reqDirect)
	if recDirect.Code != http.StatusUnauthorized {
		t.Fatalf("direct expired auditor GET = %d, want 401", recDirect.Code)
	}

	// Short-lived token expires in real time (sub-second).
	shortExp := time.Now().UTC().Add(40 * time.Millisecond)
	shortTok, err := Issue(Claims{
		Subject:          "ext-aud-short",
		Role:             RoleAdmin,
		SupplierID:       "sup-1",
		SupplierRole:     Role(SupplierRoleAuditor),
		AuditorExpiresAt: &shortExp,
	}, IssueOptions{Secret: secret, TTL: time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(60 * time.Millisecond)
	reqShort := httptest.NewRequest(http.MethodGet, "/v1/finance/packs", nil)
	reqShort.Header.Set("Authorization", "Bearer "+shortTok)
	recShort := httptest.NewRecorder()
	handler.ServeHTTP(recShort, reqShort)
	if recShort.Code != http.StatusUnauthorized {
		t.Fatalf("short-lived expired auditor GET = %d body=%s, want 401", recShort.Code, recShort.Body.String())
	}
}

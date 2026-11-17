package auth

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestJWTRoundTrip_ActorTypeAndSupplierPerms(t *testing.T) {
	t.Parallel()
	const secret = "test-supplier-rbac-jwt-secret-key-2026"
	in := Claims{
		Subject:       "usr-fin-1",
		Role:          RoleAdmin,
		SupplierID:    "sup-1",
		SupplierRole:  Role(SupplierRoleFinanceManager),
		ActorType:     ActorTypeHuman,
		SupplierPerms: []string{PermJournalApprove, PermPayoutRelease, PermAuditRead},
		MarketCode:    "UZ",
		HomeCell:      "cell-uz",
	}
	tok, err := Issue(in, IssueOptions{Secret: secret, TTL: time.Hour})
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}
	out, err := Parse(tok, secret)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if out.ActorType != ActorTypeHuman {
		t.Fatalf("ActorType = %q, want %q", out.ActorType, ActorTypeHuman)
	}
	if len(out.SupplierPerms) != 3 || out.SupplierPerms[0] != PermJournalApprove {
		t.Fatalf("SupplierPerms = %v, want %v", out.SupplierPerms, in.SupplierPerms)
	}
	if out.SupplierRole != Role(SupplierRoleFinanceManager) {
		t.Fatalf("SupplierRole = %q, want %q", out.SupplierRole, SupplierRoleFinanceManager)
	}
}

func TestSupplierRBAC_NineTemplatesAnd23Perms(t *testing.T) {
	t.Parallel()
	roles := AllSupplierRoles()
	if len(roles) != 9 {
		t.Fatalf("len(AllSupplierRoles) = %d, want 9", len(roles))
	}
	perms := AllSupplierPerms()
	if len(perms) != 23 {
		t.Fatalf("len(AllSupplierPerms) = %d, want 23", len(perms))
	}
	templates := SupplierRoleTemplates()
	if len(templates[SupplierRoleOwner]) != 23 {
		t.Fatalf("OWNER perms count = %d, want 23", len(templates[SupplierRoleOwner]))
	}
	if len(templates[SupplierRoleAuditor]) != 1 || templates[SupplierRoleAuditor][0] != PermAuditRead {
		t.Fatalf("AUDITOR perms = %v, want [audit.read]", templates[SupplierRoleAuditor])
	}
	for _, p := range templates[SupplierRoleAgent] {
		if IsAgentForbiddenPerm(p) {
			t.Fatalf("AGENT template contains forbidden perm %q", p)
		}
	}
}

func TestSupplierRBAC_BackwardCompatAndRoleInvariants(t *testing.T) {
	t.Parallel()

	// Legacy single-tenant ADMIN lacking SupplierRole resolves to OWNER with all 23 perms.
	legacyAdmin := Claims{Subject: "adm-legacy", Role: RoleAdmin, SupplierID: "sup-1"}
	if got := EffectiveSupplierRole(legacyAdmin); got != SupplierRoleOwner {
		t.Fatalf("EffectiveSupplierRole(legacyAdmin) = %q, want OWNER", got)
	}
	if !HasSupplierPerm(legacyAdmin, PermPayoutRelease) || !HasSupplierPerm(legacyAdmin, PermIAMManage) {
		t.Fatal("legacy ADMIN must retain OWNER permissions")
	}

	// AUDITOR is strictly read-only even if SupplierPerms tries to escalate.
	auditor := Claims{
		Subject:       "aud-1",
		Role:          RoleAdmin,
		SupplierID:    "sup-1",
		SupplierRole:  Role(SupplierRoleAuditor),
		SupplierPerms: []string{PermJournalPost, PermPayoutRelease, PermAuditRead},
	}
	if !IsReadOnlyAuditor(auditor) {
		t.Fatal("expected IsReadOnlyAuditor=true")
	}
	if !HasSupplierPerm(auditor, PermAuditRead) {
		t.Fatal("AUDITOR must have audit.read")
	}
	for _, forbidden := range []string{PermJournalPost, PermPayoutRelease, PermAuditManage, PermIAMManage, PermDraftApprove} {
		if HasSupplierPerm(auditor, forbidden) {
			t.Fatalf("AUDITOR must not have %q", forbidden)
		}
	}
	if got := ListSupplierPerms(auditor); len(got) != 1 || got[0] != PermAuditRead {
		t.Fatalf("ListSupplierPerms(auditor) = %v, want [audit.read]", got)
	}

	// AGENT actor (via SupplierRole=AGENT or ActorType=agent) never holds approvals/releases/reversals/period.*.
	agentByRole := Claims{
		Subject:       "agt-1",
		Role:          RoleAdmin,
		SupplierID:    "sup-1",
		SupplierRole:  Role(SupplierRoleAgent),
		SupplierPerms: []string{PermDraftApprove, PermPayoutRelease, PermJournalReverse, PermPeriodClose, PermAuditRead},
	}
	agentByType := Claims{
		Subject:       "agt-2",
		Role:          RoleAdmin,
		SupplierID:    "sup-1",
		ActorType:     ActorTypeAgent,
		SupplierPerms: []string{PermStockAdjustApprove, PermWriteoffApprove, PermCreditNoteApprove, PermAuditRead},
	}
	for _, c := range []Claims{agentByRole, agentByType} {
		if !IsAgentActor(c) {
			t.Fatalf("expected IsAgentActor=true for %+v", c)
		}
		for _, p := range []string{
			PermDraftApprove, PermCreditNoteApprove, PermStockAdjustApprove, PermWriteoffApprove,
			PermAgentProposalApprove, PermJournalApprove, PermOrderOverLimitApprove,
			PermPayoutRelease, PermJournalReverse, PermPeriodClose, PermPeriodReopen,
			PermIAMManage, PermAuditManage, PermConfigManage, PermCreditLimitChange,
		} {
			if HasSupplierPerm(c, p) {
				t.Fatalf("AGENT %s must never hold %q", c.Subject, p)
			}
		}
		if !HasSupplierPerm(c, PermAuditRead) {
			t.Fatalf("AGENT %s should retain non-privileged audit.read", c.Subject)
		}
	}

	// Non-supplier roles (RETAILER, DRIVER) never get supplier perms.
	if HasSupplierPerm(Claims{Subject: "ret-1", Role: RoleRetailer, SupplierID: "sup-1"}, PermAuditRead) {
		t.Fatal("RETAILER must not have supplier perms")
	}

	// Finance vs non-finance PII role check.
	for _, r := range []string{"ADMIN", "PLATFORM_ADMIN", "OWNER", "FINANCE_MANAGER", "ACCOUNTANT"} {
		if !IsFinanceOrAdminRole(r) {
			t.Fatalf("expected IsFinanceOrAdminRole(%q) = true", r)
		}
	}
	for _, r := range []string{"AUDITOR", "AGENT", "WAREHOUSE_MANAGER", "DISPATCHER", "RETAILER"} {
		if IsFinanceOrAdminRole(r) {
			t.Fatalf("expected IsFinanceOrAdminRole(%q) = false", r)
		}
	}
}

func TestRequireSupplierPerm_HTTPResponses(t *testing.T) {
	t.Parallel()
	okHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mw := RequireSupplierPerm(PermPayoutRelease)(okHandler)

	// 1. Unauthenticated -> 401
	recUnauth := httptest.NewRecorder()
	mw.ServeHTTP(recUnauth, httptest.NewRequest(http.MethodPost, "/v1/supplier/payouts/batches", nil))
	if recUnauth.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated status = %d, want 401", recUnauth.Code)
	}

	// 2. ACCOUNTANT lacking payout.release -> 403 JSON missing_permission
	reqAcc := httptest.NewRequest(http.MethodPost, "/v1/supplier/payouts/batches", nil)
	reqAcc = reqAcc.WithContext(WithClaims(reqAcc.Context(), Claims{
		Subject:      "acc-1",
		Role:         RoleAdmin,
		SupplierID:   "sup-1",
		SupplierRole: Role(SupplierRoleAccountant),
	}))
	recAcc := httptest.NewRecorder()
	mw.ServeHTTP(recAcc, reqAcc)
	if recAcc.Code != http.StatusForbidden {
		t.Fatalf("accountant status = %d, want 403", recAcc.Code)
	}
	var errBody map[string]string
	if err := json.Unmarshal(recAcc.Body.Bytes(), &errBody); err != nil {
		t.Fatalf("decode 403 JSON: %v", err)
	}
	if errBody["detail"] != "missing_permission" || errBody["permission"] != PermPayoutRelease {
		t.Fatalf("unexpected 403 JSON: %+v", errBody)
	}

	// 3. FINANCE_MANAGER holding payout.release -> 200 OK
	reqFM := httptest.NewRequest(http.MethodPost, "/v1/supplier/payouts/batches", nil)
	reqFM = reqFM.WithContext(WithClaims(reqFM.Context(), Claims{
		Subject:      "fm-1",
		Role:         RoleAdmin,
		SupplierID:   "sup-1",
		SupplierRole: Role(SupplierRoleFinanceManager),
	}))
	recFM := httptest.NewRecorder()
	mw.ServeHTTP(recFM, reqFM)
	if recFM.Code != http.StatusOK {
		t.Fatalf("finance_manager status = %d, want 200", recFM.Code)
	}
}

func makeTestCertificatePEM(t *testing.T, notBefore, notAfter time.Time) (string, *x509.Certificate) {
	t.Helper()
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("rsa.GenerateKey: %v", err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "terminal-payloader-1"},
		NotBefore:    notBefore,
		NotAfter:     notAfter,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &priv.PublicKey, priv)
	if err != nil {
		t.Fatalf("x509.CreateCertificate: %v", err)
	}
	parsed, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatalf("x509.ParseCertificate: %v", err)
	}
	pemBytes := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	return string(pemBytes), parsed
}

func TestRequireDeviceCert_HardenedProduction(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	h := RequireDeviceCert(next)

	// Non-production allows missing cert.
	t.Setenv("PEGASUSX_ENV", "local")
	recLocal := httptest.NewRecorder()
	h.ServeHTTP(recLocal, httptest.NewRequest(http.MethodPost, "/v1/auth/payloader/login", nil))
	if recLocal.Code != http.StatusOK {
		t.Fatalf("local status = %d, want 200", recLocal.Code)
	}

	// Production behind TLS-terminating LB (r.TLS == nil) without headers -> 401!
	t.Setenv("PEGASUSX_ENV", "production")
	recNilTLS := httptest.NewRecorder()
	h.ServeHTTP(recNilTLS, httptest.NewRequest(http.MethodPost, "/v1/auth/payloader/login", nil))
	if recNilTLS.Code != http.StatusUnauthorized || !strings.Contains(recNilTLS.Body.String(), "hardware_attestation_required") {
		t.Fatalf("prod r.TLS==nil without headers got (%d, %s), want 401", recNilTLS.Code, recNilTLS.Body.String())
	}

	// Placeholder header rejected.
	reqMock := httptest.NewRequest(http.MethodPost, "/v1/auth/payloader/login", nil)
	reqMock.Header.Set("X-Device-Cert", "mock")
	recMock := httptest.NewRecorder()
	h.ServeHTTP(recMock, reqMock)
	if recMock.Code != http.StatusUnauthorized {
		t.Fatalf("prod placeholder X-Device-Cert got %d, want 401", recMock.Code)
	}

	// Opaque arbitrary string rejected even from trusted proxy.
	t.Setenv("TRUSTED_PROXY_CIDRS", "192.0.2.0/24")
	reqOpaque := httptest.NewRequest(http.MethodPost, "/v1/auth/payloader/login", nil)
	reqOpaque.Header.Set("X-Device-Cert", "1234567890abcdef")
	recOpaque := httptest.NewRecorder()
	h.ServeHTTP(recOpaque, reqOpaque)
	if recOpaque.Code != http.StatusUnauthorized {
		t.Fatalf("prod opaque X-Device-Cert got %d, want 401", recOpaque.Code)
	}

	// Untrusted proxy IP with XFCC / X-Device-Cert rejected even if cert valid.
	validPEM, validCert := makeTestCertificatePEM(t, time.Now().Add(-time.Hour), time.Now().Add(time.Hour))
	reqUntrusted := httptest.NewRequest(http.MethodPost, "/v1/auth/payloader/login", nil)
	reqUntrusted.RemoteAddr = "198.51.100.77:5555"
	reqUntrusted.Header.Set("X-Device-Cert", validPEM)
	recUntrusted := httptest.NewRecorder()
	h.ServeHTTP(recUntrusted, reqUntrusted)
	if recUntrusted.Code != http.StatusUnauthorized {
		t.Fatalf("untrusted remote proxy X-Device-Cert got %d, want 401", recUntrusted.Code)
	}

	// Valid X-Forwarded-Client-Cert accepted when r.TLS == nil from trusted proxy.
	reqXFCC := httptest.NewRequest(http.MethodPost, "/v1/auth/payloader/login", nil)
	reqXFCC.Header.Set("X-Forwarded-Client-Cert", `By=spiffe://cluster.local/ns/prod/sa/gw;Hash=468ed33be74eee6556d90c0149c1309e9ba61d6425303443c0748a02dd8de688;Subject="CN=terminal-1"`)
	recXFCC := httptest.NewRecorder()
	h.ServeHTTP(recXFCC, reqXFCC)
	if recXFCC.Code != http.StatusOK {
		t.Fatalf("prod valid XFCC got %d, want 200", recXFCC.Code)
	}

	// Valid PEM X-Device-Cert accepted from trusted proxy; expired PEM rejected.
	reqPEM := httptest.NewRequest(http.MethodPost, "/v1/auth/payloader/login", nil)
	reqPEM.Header.Set("X-Device-Cert", validPEM)
	recPEM := httptest.NewRecorder()
	h.ServeHTTP(recPEM, reqPEM)
	if recPEM.Code != http.StatusOK {
		t.Fatalf("prod valid PEM X-Device-Cert got %d, want 200", recPEM.Code)
	}

	// When DEVICE_CERT_CA_PEM is configured, self-signed untrusted certs must be rejected.
	t.Setenv("DEVICE_CERT_CA_PEM", func() string {
		otherPEM, _ := makeTestCertificatePEM(t, time.Now().Add(-time.Hour), time.Now().Add(time.Hour))
		return otherPEM
	}())
	recWrongCA := httptest.NewRecorder()
	h.ServeHTTP(recWrongCA, reqPEM)
	if recWrongCA.Code != http.StatusUnauthorized {
		t.Fatalf("prod wrong CA X-Device-Cert got %d, want 401", recWrongCA.Code)
	}
	t.Setenv("DEVICE_CERT_CA_PEM", validPEM)
	recMatchCA := httptest.NewRecorder()
	h.ServeHTTP(recMatchCA, reqPEM)
	if recMatchCA.Code != http.StatusOK {
		t.Fatalf("prod matching CA X-Device-Cert got %d, want 200", recMatchCA.Code)
	}
	t.Setenv("DEVICE_CERT_CA_PEM", "")

	expiredPEM, _ := makeTestCertificatePEM(t, time.Now().Add(-48*time.Hour), time.Now().Add(-24*time.Hour))
	reqExp := httptest.NewRequest(http.MethodPost, "/v1/auth/payloader/login", nil)
	reqExp.Header.Set("X-Device-Cert", expiredPEM)
	recExp := httptest.NewRecorder()
	h.ServeHTTP(recExp, reqExp)
	if recExp.Code != http.StatusUnauthorized {
		t.Fatalf("prod expired PEM got %d, want 401", recExp.Code)
	}

	// Valid r.TLS.PeerCertificates accepted.
	reqPeer := httptest.NewRequest(http.MethodPost, "/v1/auth/payloader/login", nil)
	reqPeer.TLS = &tls.ConnectionState{PeerCertificates: []*x509.Certificate{validCert}}
	recPeer := httptest.NewRecorder()
	h.ServeHTTP(recPeer, reqPeer)
	if recPeer.Code != http.StatusOK {
		t.Fatalf("prod valid PeerCertificates got %d, want 200", recPeer.Code)
	}
}

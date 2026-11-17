package partner

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"net"
	"testing"
	"time"

	"golang.org/x/crypto/ssh"
)

func TestSftpHostKeyCallbackPin(t *testing.T) {
	key := &fakeHostKey{blob: []byte("pegasusx-host-key-material")}
	sum := sha256.Sum256(key.Marshal())
	pin := base64.RawStdEncoding.EncodeToString(sum[:])
	cb, err := sftpHostKeyCallback(SftpConfig{HostKeySHA256: pin})
	if err != nil {
		t.Fatal(err)
	}
	if err := cb("host:22", fakeAddr{}, key); err != nil {
		t.Fatalf("expected match: %v", err)
	}
	bad := &fakeHostKey{blob: []byte("other")}
	if err := cb("host:22", fakeAddr{}, bad); err == nil {
		t.Fatal("expected mismatch")
	}
}

func TestSftpHostKeyStrictRequiresPin(t *testing.T) {
	t.Setenv("PARTNER_SFTP_STRICT_HOSTKEY", "true")
	_, err := sftpHostKeyCallback(SftpConfig{})
	if err == nil {
		t.Fatal("expected host key required")
	}
}

func TestRotateWebhookSecret(t *testing.T) {
	keys := NewMemoryKeyRepository()
	hooks := NewMemoryWebhookRepository()
	svc := NewService(keys, hooks, nil, nil, nil)
	svc.WebhookURLPolicy = &WebhookURLPolicy{HostAllowlist: []string{"example.test"}}
	p := Principal{TenantType: TenantSupplier, TenantID: "sup-1", Scopes: []string{ScopeWebhooksManage}}
	sub, secret, err := svc.CreateWebhookSubscription(context.Background(), p, "https://example.test/hook", []string{"ORDER_CREATED"})
	if err != nil {
		t.Fatal(err)
	}
	if secret == "" {
		t.Fatal("empty secret")
	}
	next, err := svc.RotateWebhookSecret(context.Background(), p, sub.SubscriptionID)
	if err != nil {
		t.Fatal(err)
	}
	if next == "" || next == secret {
		t.Fatalf("rotate did not change secret")
	}
}

func TestAckAcceptedFromExternal(t *testing.T) {
	if !ackAcceptedFromExternal("PO-1:CONTRL:OK") {
		t.Fatal("expected OK")
	}
	if ackAcceptedFromExternal("PO-1:APERAK:REJ:bad") {
		t.Fatal("expected REJ")
	}
	_ = time.Now()
}

type fakeHostKey struct{ blob []byte }

func (f *fakeHostKey) Type() string                { return "ssh-ed25519" }
func (f *fakeHostKey) Marshal() []byte             { return f.blob }
func (f *fakeHostKey) Verify([]byte, *ssh.Signature) error { return nil }

type fakeAddr struct{}

func (fakeAddr) Network() string { return "tcp" }
func (fakeAddr) String() string  { return "127.0.0.1:22" }

var _ net.Addr = fakeAddr{}
var _ ssh.PublicKey = (*fakeHostKey)(nil)

func TestPartnerSecurityHardening(t *testing.T) {
	// 1. LoadSecretRef blocks arbitrary env vars and filesystem paths.
	t.Setenv("ARBITRARY_JWT_SECRET", "top-secret-jwt")
	if _, err := LoadSecretRef("ARBITRARY_JWT_SECRET"); err == nil {
		t.Fatal("LoadSecretRef must reject unprefixed OS env var")
	}
	if _, err := LoadSecretRef("/etc/passwd"); err == nil {
		t.Fatal("LoadSecretRef must reject filesystem paths")
	}
	if _, err := LoadSecretRef("./local.key"); err == nil {
		t.Fatal("LoadSecretRef must reject relative paths")
	}
	t.Setenv("PARTNER_SFTP_SECRET_SUPPLIER_SUP_1_SFTP1", "tenant-secret-val")
	t.Setenv("PARTNER_SFTP_SECRET_SFTP1", "shared-secret-val")
	if got, err := LoadTenantSecretRef("SUPPLIER", "sup-1", "sftp1"); err != nil || got != "tenant-secret-val" {
		t.Fatalf("expected tenant-scoped secret, got=%q err=%v", got, err)
	}
	t.Setenv("PEGASUSX_ENV", "production")
	if _, err := LoadTenantSecretRef("SUPPLIER", "sup-2", "sftp1"); err == nil {
		t.Fatal("expected fail-closed in production when tenant sup-2 attempts cross-tenant sftp1 ref")
	}
	t.Setenv("PEGASUSX_ENV", "")

	// 2. Host-key pinning mandatory by default even without env var set.
	t.Setenv("PARTNER_SFTP_STRICT_HOSTKEY", "")
	t.Setenv("PARTNER_SFTP_ALLOW_INSECURE_HOSTKEY", "")
	if _, err := sftpHostKeyCallback(SftpConfig{}); err == nil {
		t.Fatal("expected sftpHostKeyCallback to require host key by default")
	}

	// 3. SFTP directory traversal rejected.
	if err := ValidateSftpConfigPaths(SftpConfig{RemoteDir: "/edi", InboundDir: "../etc", OutboundDir: "out", ArchiveDir: "arch"}); err == nil {
		t.Fatal("expected traversal rejection on InboundDir")
	}

	// 4. DeliveryWorker blocks SSRF by default.
	dw := NewDeliveryWorker(NewMemoryWebhookRepository(), nil)
	if err := dw.DeliverHTTP(context.Background(), WebhookSubscription{URL: "https://169.254.169.254/latest/meta-data/", SigningSecret: "s"}, "e1", "ORDER_CREATED", []byte(`{}`)); err == nil {
		t.Fatal("expected DeliveryWorker SSRF block")
	}

	// 5. CSV and XLSX formula injection neutralization (preserving negative numbers).
	rows := []map[string]any{
		{"sku": "=CMD|'/C calc'!A0", "delta": -42, "note": "  +SUM(A1:A2)"},
	}
	csvBytes, _, err := encodeExportCSV(rows)
	if err != nil {
		t.Fatal(err)
	}
	csvStr := string(csvBytes)
	if !containsAll(csvStr, "'=CMD|", "-42,", "'  +SUM(A1:A2)") {
		t.Fatalf("CSV formula sanitization unexpected: %s", csvStr)
	}
}

func containsAll(hay string, needles ...string) bool {
	for _, n := range needles {
		found := false
		for i := 0; i+len(n) <= len(hay); i++ {
			if hay[i:i+len(n)] == n {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

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

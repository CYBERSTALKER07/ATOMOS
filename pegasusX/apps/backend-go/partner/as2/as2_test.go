package as2

import (
	"net/http"
	"strings"
	"testing"
)

func TestSignEncryptRoundTrip(t *testing.T) {
	our, err := GenerateSelfSignedRSA("our")
	if err != nil {
		t.Fatal(err)
	}
	partner, err := GenerateSelfSignedRSA("partner")
	if err != nil {
		t.Fatal(err)
	}
	edi := []byte("UNA:+.? 'UNB+AS2'test'")
	enveloped, err := SignThenEncrypt(edi, partner, our.Cert)
	if err != nil {
		t.Fatal(err)
	}
	out, err := DecryptThenVerify(enveloped, our, partner.Cert)
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != string(edi) {
		t.Fatalf("got %q want %q", out, edi)
	}
}

func TestMICStable(t *testing.T) {
	a := MICSHA256([]byte("hello"))
	b := MICSHA256([]byte("hello"))
	if a != b || !strings.HasSuffix(a, ", sha-256") {
		t.Fatalf("mic=%q", a)
	}
}

func TestUnwrapPlain(t *testing.T) {
	h := http.Header{}
	h.Set("Content-Type", ContentTypeEDIFACT)
	h.Set("Content-Disposition", `attachment; filename="ORDERS_x.edi"`)
	body := []byte("UNA:+.? 'UNH+1+ORDERS'")
	p, err := UnwrapInbound(h, body, Material{}, nil, true)
	if err != nil {
		t.Fatal(err)
	}
	if string(p.Content) != string(body) {
		t.Fatalf("content mismatch")
	}
	if p.Filename != "ORDERS_x.edi" {
		t.Fatalf("filename=%q", p.Filename)
	}
	if p.MIC == "" {
		t.Fatal("missing mic")
	}
}

func TestSyncMDNContainsMIC(t *testing.T) {
	ct, body, err := BuildSyncMDN(MessageHeaders{
		AS2From: "PARTNER", AS2To: "US", MessageID: "<m1@x>",
	}, MICSHA256([]byte("edi")), MDNProcessed, "")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(ct, "disposition-notification") {
		t.Fatalf("ct=%s", ct)
	}
	if !strings.Contains(string(body), "Received-Content-MIC:") {
		t.Fatalf("body=%s", body)
	}
	if !strings.Contains(string(body), "processed") {
		t.Fatalf("body missing processed")
	}
}

func TestUnwrapEncrypted(t *testing.T) {
	our, err := GenerateSelfSignedRSA("our")
	if err != nil {
		t.Fatal(err)
	}
	partner, err := GenerateSelfSignedRSA("partner")
	if err != nil {
		t.Fatal(err)
	}
	edi := []byte("UNA:+.? 'UNH+1+ORDERS:D:96A:UN'")
	ct, body, err := BuildEncryptedBody(edi, partner, EncodeCertPEM(our.Cert), "smime.p7m")
	if err != nil {
		t.Fatal(err)
	}
	h := http.Header{}
	h.Set("Content-Type", ct)
	p, err := UnwrapInbound(h, body, our, EncodeCertPEM(partner.Cert), false)
	if err != nil {
		t.Fatal(err)
	}
	if string(p.Content) != string(edi) {
		t.Fatalf("got %q", p.Content)
	}
}

package as2

import (
	"context"
	"net/http"
	"net/http/httptest"
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

func TestParseAndVerifyMDN(t *testing.T) {
	edi := []byte("UNA:+.? 'UNH+1+ORDERS'")
	mic := MICSHA256(edi)
	ct, body, err := BuildSyncMDN(MessageHeaders{
		AS2From: "P", AS2To: "U", MessageID: "<m@x>",
	}, mic, MDNProcessed, "")
	if err != nil {
		t.Fatal(err)
	}
	mdn, err := ParseSyncMDN(ct, body)
	if err != nil {
		t.Fatal(err)
	}
	if err := VerifyMDN(mic, mdn); err != nil {
		t.Fatal(err)
	}
	if err := VerifyMDN(MICSHA256([]byte("other")), mdn); err == nil {
		t.Fatal("expected mic mismatch")
	}
	bad := mdn
	bad.Disposition = string(MDNFailed)
	if err := VerifyMDN(mic, bad); err == nil {
		t.Fatal("expected failed disposition")
	}
}

func TestSendVerifiesMDN(t *testing.T) {
	edi := []byte("UNA:+.? 'UNH+1+DESADV'")
	mic := MICSHA256(edi)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ct, body, err := BuildSyncMDN(MessageHeaders{
			AS2From: "PARTNER", AS2To: "US", MessageID: r.Header.Get(HeaderMessageID),
		}, mic, MDNProcessed, "")
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		w.Header().Set("Content-Type", ct)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
	}))
	defer ts.Close()

	c := NewClient()
	res, err := c.Send(context.Background(), SendRequest{
		URL: ts.URL, From: "US", To: "PARTNER", EDI: edi, Plain: true, RequestMDN: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.MDN.ReceivedContentMIC == "" {
		t.Fatal("missing parsed MIC")
	}

	tsBad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ct, body, err := BuildSyncMDN(MessageHeaders{
			AS2From: "PARTNER", AS2To: "US", MessageID: "<x>",
		}, MICSHA256([]byte("tampered")), MDNProcessed, "")
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		w.Header().Set("Content-Type", ct)
		_, _ = w.Write(body)
	}))
	defer tsBad.Close()
	_, err = c.Send(context.Background(), SendRequest{
		URL: tsBad.URL, From: "US", To: "PARTNER", EDI: edi, Plain: true, RequestMDN: true,
	})
	if err == nil || !strings.Contains(err.Error(), "mic_mismatch") {
		t.Fatalf("want mic_mismatch, got %v", err)
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

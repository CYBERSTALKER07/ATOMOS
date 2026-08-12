package ar

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func testLogger() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

func jsonUnmarshal(raw []byte, v any) error { return json.Unmarshal(raw, v) }

// rewriteHost redirects all requests to the test server (SendGrid host is
// fixed in the transport, so tests swap the round tripper).
type rewriteHost struct{ target string }

func (r rewriteHost) RoundTrip(req *http.Request) (*http.Response, error) {
	u, err := url.Parse(r.target)
	if err != nil {
		return nil, err
	}
	req.URL.Scheme = u.Scheme
	req.URL.Host = u.Host
	return http.DefaultTransport.RoundTrip(req)
}

func TestTransportsFromEnv_FailClosed(t *testing.T) {
	t.Setenv("DUNNING_SMS_PROVIDER", "")
	t.Setenv("DUNNING_EMAIL_PROVIDER", "")
	t.Setenv("DUNNING_WHATSAPP_PROVIDER", "")
	tr, err := TransportsFromEnv()
	if err != nil || len(tr) != 0 {
		t.Fatalf("unset providers must build empty: tr=%v err=%v", tr, err)
	}

	t.Setenv("DUNNING_SMS_PROVIDER", "pigeon")
	if _, err := TransportsFromEnv(); err == nil || !strings.Contains(err.Error(), "unknown DUNNING_SMS_PROVIDER") {
		t.Fatalf("unknown SMS provider must fail, got %v", err)
	}

	t.Setenv("DUNNING_SMS_PROVIDER", "twilio")
	t.Setenv("TWILIO_ACCOUNT_SID", "")
	t.Setenv("TWILIO_AUTH_TOKEN", "")
	t.Setenv("TWILIO_FROM_NUMBER", "")
	if _, err := TransportsFromEnv(); err == nil || !strings.Contains(err.Error(), "TWILIO_ACCOUNT_SID") {
		t.Fatalf("twilio without creds must fail, got %v", err)
	}

	t.Setenv("DUNNING_SMS_PROVIDER", "")
	t.Setenv("DUNNING_EMAIL_PROVIDER", "sendgrid")
	t.Setenv("SENDGRID_API_KEY", "")
	t.Setenv("SENDGRID_FROM_EMAIL", "")
	if _, err := TransportsFromEnv(); err == nil || !strings.Contains(err.Error(), "SENDGRID_API_KEY") {
		t.Fatalf("sendgrid without creds must fail, got %v", err)
	}

	t.Setenv("DUNNING_EMAIL_PROVIDER", "")
	t.Setenv("DUNNING_WHATSAPP_PROVIDER", "carrier-pigeon")
	if _, err := TransportsFromEnv(); err == nil || !strings.Contains(err.Error(), "unknown DUNNING_WHATSAPP_PROVIDER") {
		t.Fatalf("unknown WhatsApp provider must fail, got %v", err)
	}

	t.Setenv("DUNNING_WHATSAPP_PROVIDER", "twilio")
	t.Setenv("TWILIO_ACCOUNT_SID", "AC1")
	t.Setenv("TWILIO_AUTH_TOKEN", "tok")
	t.Setenv("TWILIO_WHATSAPP_FROM", "")
	t.Setenv("TWILIO_WHATSAPP_CONTENT_SID", "")
	if _, err := TransportsFromEnv(); err == nil || !strings.Contains(err.Error(), "TWILIO_WHATSAPP_FROM") {
		t.Fatalf("twilio WhatsApp without FROM/CONTENT_SID must fail, got %v", err)
	}
}

type fakeTransport struct {
	channel string
	sent    []string
	err     error
}

func (f *fakeTransport) Channel() string { return f.channel }
func (f *fakeTransport) Send(_ context.Context, to Contact, body string) error {
	f.sent = append(f.sent, body)
	return f.err
}

type fakeResolver struct {
	retailer Contact
	staff    []Contact
}

func (f fakeResolver) ResolveRetailer(_ context.Context, _ string) (Contact, error) { return f.retailer, nil }
func (f fakeResolver) ResolveSupplierStaff(_ context.Context, _ string) ([]Contact, error) {
	return f.staff, nil
}

func TestMultiChannelNotify_FanoutAndFailureIsolation(t *testing.T) {
	sms := &fakeTransport{channel: "sms"}
	email := &fakeTransport{channel: "email", err: context.DeadlineExceeded}
	resolver := fakeResolver{
		retailer: Contact{Phone: "+998901234567"}, // no email: email channel skips retailer
		staff:    []Contact{{Email: "boss@supplier.uz", Phone: "+998909999999"}},
	}
	notify := MultiChannelNotify(testLogger(), resolver, []ChannelTransport{sms, email})
	err := notify(context.Background(), Invoice{InvoiceID: "inv-1", RetailerID: "r", SupplierID: "s", BalanceMinor: 1000}, 0, 1)
	if err == nil {
		t.Fatal("aggregated send error expected (email transport failed)")
	}
	// SMS reached both targets despite email failing.
	if len(sms.sent) != 2 {
		t.Fatalf("sms sends = %d, want 2 (retailer + staff)", len(sms.sent))
	}
	// Email only attempted for the staff contact (retailer has no email).
	if len(email.sent) != 1 {
		t.Fatalf("email sends = %d, want 1 (staff only)", len(email.sent))
	}
}

func TestTwilioSMS_Contract(t *testing.T) {
	var gotAuth, gotBody, gotPath string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		raw, _ := io.ReadAll(r.Body)
		gotBody = string(raw)
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"sid":"SM123"}`))
	}))
	defer ts.Close()

	tr := &twilioSMS{sid: "AC1", token: "tok", from: "+15005550006", baseURL: ts.URL, hc: ts.Client()}
	if err := tr.Send(context.Background(), Contact{Phone: "+998901234567"}, "pay invoice inv-1"); err != nil {
		t.Fatalf("send: %v", err)
	}
	if gotPath != "/2010-04-01/Accounts/AC1/Messages.json" {
		t.Fatalf("path = %q", gotPath)
	}
	if !strings.HasPrefix(gotAuth, "Basic ") {
		t.Fatalf("auth = %q, want Basic", gotAuth)
	}
	for _, want := range []string{"To=%2B998901234567", "From=%2B15005550006", "pay+invoice+inv-1"} {
		if !strings.Contains(gotBody, want) {
			t.Fatalf("form %q missing %q", gotBody, want)
		}
	}
}

func TestSendGridEmail_Contract(t *testing.T) {
	var gotAuth string
	var gotPayload map[string]any
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		raw, _ := io.ReadAll(r.Body)
		_ = jsonUnmarshal(raw, &gotPayload)
		w.WriteHeader(http.StatusAccepted)
	}))
	defer ts.Close()

	// sendGridEmail has a fixed host; exercise via a transport that rewrites to ts.URL.
	sg := &sendGridEmail{apiKey: "SG.KEY", from: "ar@pegasusx.uz", fromName: "PegasusX Collections", hc: &http.Client{
		Transport: rewriteHost{target: ts.URL},
		Timeout:   0,
	}}
	if err := sg.Send(context.Background(), Contact{Email: "boss@supplier.uz", Name: "Boss"}, "invoice inv-1 due"); err != nil {
		t.Fatalf("send: %v", err)
	}
	if gotAuth != "Bearer SG.KEY" {
		t.Fatalf("auth = %q", gotAuth)
	}
	to := gotPayload["personalizations"].([]any)[0].(map[string]any)["to"].([]any)[0].(map[string]any)
	if to["email"] != "boss@supplier.uz" {
		t.Fatalf("to = %v", to)
	}
	if gotPayload["from"].(map[string]any)["email"] != "ar@pegasusx.uz" {
		t.Fatalf("from = %v", gotPayload["from"])
	}
}

func TestPlayMobileSMS_Contract(t *testing.T) {
	var gotAuth string
	var gotPayload map[string]any
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		raw, _ := io.ReadAll(r.Body)
		_ = jsonUnmarshal(raw, &gotPayload)
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	pm := &playMobileSMS{login: "l", password: "p", baseURL: ts.URL, hc: ts.Client()}
	if err := pm.Send(context.Background(), Contact{Phone: "+998901234567"}, "due 1000 UZS"); err != nil {
		t.Fatalf("send: %v", err)
	}
	if !strings.HasPrefix(gotAuth, "Basic ") {
		t.Fatalf("auth = %q", gotAuth)
	}
	msg := gotPayload["messages"].([]any)[0].(map[string]any)
	if msg["recipient"] != "998901234567" {
		t.Fatalf("recipient = %v (plus must be stripped)", msg["recipient"])
	}
	if msg["sms"].(map[string]any)["content"] != "due 1000 UZS" {
		t.Fatalf("content = %v", msg["sms"])
	}
}

func TestTwilioWhatsApp_Contract(t *testing.T) {
	var gotAuth, gotBody, gotPath string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		raw, _ := io.ReadAll(r.Body)
		gotBody = string(raw)
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"sid":"SMWA"}`))
	}))
	defer ts.Close()

	tr := &twilioWhatsApp{
		sid: "AC1", token: "tok", from: "+15005550006",
		contentSID: "HXabc", contentVarBody: "1",
		baseURL: ts.URL, hc: ts.Client(),
	}
	if err := tr.Send(context.Background(), Contact{Phone: "+998901234567"}, "pay invoice inv-1"); err != nil {
		t.Fatalf("send: %v", err)
	}
	if gotPath != "/2010-04-01/Accounts/AC1/Messages.json" {
		t.Fatalf("path = %q", gotPath)
	}
	if !strings.HasPrefix(gotAuth, "Basic ") {
		t.Fatalf("auth = %q, want Basic", gotAuth)
	}
	for _, want := range []string{
		"To=whatsapp%3A%2B998901234567",
		"From=whatsapp%3A%2B15005550006",
		"ContentSid=HXabc",
		"ContentVariables=",
	} {
		if !strings.Contains(gotBody, want) {
			t.Fatalf("form %q missing %q", gotBody, want)
		}
	}
	if !strings.Contains(gotBody, "pay") {
		t.Fatalf("form %q missing notice body in ContentVariables", gotBody)
	}
}

func TestTwilioWhatsApp_PrefixedFromPassthrough(t *testing.T) {
	var gotBody string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		gotBody = string(raw)
		w.WriteHeader(http.StatusCreated)
	}))
	defer ts.Close()

	tr := &twilioWhatsApp{
		sid: "AC1", token: "tok", from: "whatsapp:+15005550006",
		contentSID: "HXabc", contentVarBody: "1",
		baseURL: ts.URL, hc: ts.Client(),
	}
	if err := tr.Send(context.Background(), Contact{Phone: "whatsapp:+998901234567"}, "x"); err != nil {
		t.Fatalf("send: %v", err)
	}
	if !strings.Contains(gotBody, "From=whatsapp%3A%2B15005550006") {
		t.Fatalf("From should keep single whatsapp: prefix, got %q", gotBody)
	}
}

func TestTransportErrorStatus_Surfaces(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"message":"bad creds"}`))
	}))
	defer ts.Close()

	tr := &twilioSMS{sid: "AC1", token: "bad", from: "+1", baseURL: ts.URL, hc: ts.Client()}
	err := tr.Send(context.Background(), Contact{Phone: "+998901234567"}, "x")
	if err == nil || !strings.Contains(err.Error(), "401") {
		t.Fatalf("want 401 surfaced, got %v", err)
	}
}

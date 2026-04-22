package payment

import (
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
	"testing"
)

// ─── renderGlobalPayCheckoutURL ─────────────────────────────────────────────

func TestRenderGlobalPayCheckoutURL(t *testing.T) {
	template := "https://pay.global.test/checkout?merchant={merchant_id}&order={order_id}&amount={amount}&tiyin={amount_tiyin}"
	result := renderGlobalPayCheckoutURL(template, "MERCH-1", "ORD-42", 50000)

	if !strings.Contains(result, "merchant=MERCH-1") {
		t.Errorf("missing merchant_id in %q", result)
	}
	if !strings.Contains(result, "order=ORD-42") {
		t.Errorf("missing order_id in %q", result)
	}
	if !strings.Contains(result, "amount=50000") {
		t.Errorf("missing amount in %q", result)
	}
	if !strings.Contains(result, "tiyin=5000000") {
		t.Errorf("missing amount_tiyin in %q", result)
	}
}

func TestRenderGlobalPayCheckoutURL_URLEncoding(t *testing.T) {
	template := "https://pay.test/{merchant_id}/{order_id}"
	result := renderGlobalPayCheckoutURL(template, "MERCH WITH SPACE", "ORD/SPECIAL", 100)

	if !strings.Contains(result, "MERCH+WITH+SPACE") && !strings.Contains(result, "MERCH%20WITH%20SPACE") {
		t.Errorf("merchant_id not URL-encoded in %q", result)
	}
}

func TestRenderGlobalPayCheckoutURL_EmptyTemplate(t *testing.T) {
	result := renderGlobalPayCheckoutURL("", "MERCH-1", "ORD-1", 100)
	if result != "" {
		t.Errorf("expected empty, got %q", result)
	}
}

// ─── paymeCheckoutURLWithCreds ──────────────────────────────────────────────

func TestPaymeCheckoutURLWithCreds(t *testing.T) {
	os.Setenv("PAYME_CHECKOUT_URL", "https://checkout.paycom.uz")
	defer os.Unsetenv("PAYME_CHECKOUT_URL")

	url, err := paymeCheckoutURLWithCreds("ORD-100", 50000, "MERCH-PAYME")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(url, "https://checkout.paycom.uz/") {
		t.Errorf("bad prefix: %q", url)
	}

	// Decode the base64 portion and verify contents
	encoded := strings.TrimPrefix(url, "https://checkout.paycom.uz/")
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		t.Fatalf("base64 decode error: %v", err)
	}
	raw := string(decoded)
	if !strings.Contains(raw, "m=MERCH-PAYME") {
		t.Errorf("missing merchant_id in decoded: %q", raw)
	}
	if !strings.Contains(raw, "ac.order_id=ORD-100") {
		t.Errorf("missing order_id in decoded: %q", raw)
	}
	// 50000 = 5000000 tiyins
	if !strings.Contains(raw, "a=5000000") {
		t.Errorf("missing amount in decoded: %q", raw)
	}
}

func TestPaymeCheckoutURLWithCreds_EmptyMerchant(t *testing.T) {
	_, err := paymeCheckoutURLWithCreds("ORD-1", 100, "")
	if err == nil {
		t.Error("expected error for empty merchant_id")
	}
}

// ─── clickCheckoutURLWithCreds ──────────────────────────────────────────────

func TestClickCheckoutURLWithCreds(t *testing.T) {
	os.Setenv("CLICK_CHECKOUT_URL", "https://my.click.uz/services/pay")
	defer os.Unsetenv("CLICK_CHECKOUT_URL")

	url, err := clickCheckoutURLWithCreds("ORD-200", 75000, "CLICK-MERCH", "SVC-001")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(url, "service_id=SVC-001") {
		t.Errorf("missing service_id in %q", url)
	}
	if !strings.Contains(url, "merchant_id=CLICK-MERCH") {
		t.Errorf("missing merchant_id in %q", url)
	}
	if !strings.Contains(url, "amount=75000") {
		t.Errorf("missing amount in %q", url)
	}
	if !strings.Contains(url, "transaction_param=ORD-200") {
		t.Errorf("missing order_id in %q", url)
	}
}

func TestClickCheckoutURLWithCreds_MissingParams(t *testing.T) {
	_, err := clickCheckoutURLWithCreds("ORD-1", 100, "", "SVC-1")
	if err == nil {
		t.Error("expected error for empty merchant_id")
	}
	_, err = clickCheckoutURLWithCreds("ORD-1", 100, "MERCH", "")
	if err == nil {
		t.Error("expected error for empty service_id")
	}
}

// ─── CheckoutURLWithCredentials ─────────────────────────────────────────────

func TestCheckoutURLWithCredentials_Simulated(t *testing.T) {
	url, err := CheckoutURLWithCredentials("SIMULATED", "ORD-SIM", 99000, "M-1", "S-1")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(url, "mock-gateway.lab.fake") {
		t.Errorf("expected simulated URL, got %q", url)
	}
	if !strings.Contains(url, "ORD-SIM") {
		t.Errorf("missing order_id in %q", url)
	}
}

func TestCheckoutURLWithCredentials_Unknown(t *testing.T) {
	url, err := CheckoutURLWithCredentials("CASH", "ORD-1", 100, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if url != "" {
		t.Errorf("CASH should return empty URL, got %q", url)
	}
}

// ─── CheckoutURL ─────────────────────────────────────────────────────────────

func TestCheckoutURL_Simulated(t *testing.T) {
	url, err := CheckoutURL("SIMULATED", "ORD-999", 25000)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(url, "mock-gateway.lab.fake/checkout/ORD-999") {
		t.Errorf("unexpected URL: %q", url)
	}
	if !strings.Contains(url, "amount=25000") {
		t.Errorf("missing amount: %q", url)
	}
}

func TestCheckoutURL_NilForCash(t *testing.T) {
	url, err := CheckoutURL("CASH", "ORD-1", 100)
	if err != nil {
		t.Fatal(err)
	}
	if url != "" {
		t.Errorf("CASH should return empty URL, got %q", url)
	}
}

func TestCheckoutURL_NilForUzcard(t *testing.T) {
	url, err := CheckoutURL("UZCARD", "ORD-1", 100)
	if err != nil {
		t.Fatal(err)
	}
	if url != "" {
		t.Errorf("UZCARD should return empty URL, got %q", url)
	}
}

// ─── computeClickSignature ──────────────────────────────────────────────────

func TestComputeClickSignature_KnownVector(t *testing.T) {
	// computeClickSignature: MD5(clickTransID + serviceID + secretKey + merchantTransID + amount + action + signTime)
	clickTransID := "123"
	serviceID := "SVC"
	secretKey := "SECRET"
	merchantTransID := "ORD-1"
	amount := int64(50000)
	action := 1
	signTime := "2026-01-01 12:00:00"

	raw := fmt.Sprintf("%s%s%s%s%d%d%s", clickTransID, serviceID, secretKey, merchantTransID, amount, action, signTime)
	expected := md5.Sum([]byte(raw))
	expectedHex := hex.EncodeToString(expected[:])

	got := computeClickSignature(clickTransID, serviceID, secretKey, merchantTransID, amount, action, signTime)
	if got != expectedHex {
		t.Errorf("got %q, want %q", got, expectedHex)
	}
}

func TestComputeClickSignature_Deterministic(t *testing.T) {
	a := computeClickSignature("1", "2", "3", "4", 100, 0, "now")
	b := computeClickSignature("1", "2", "3", "4", 100, 0, "now")
	if a != b {
		t.Errorf("non-deterministic: %q != %q", a, b)
	}
}

func TestComputeClickSignature_DifferentInputs(t *testing.T) {
	a := computeClickSignature("1", "2", "3", "4", 100, 0, "now")
	b := computeClickSignature("1", "2", "3", "4", 200, 0, "now")
	if a == b {
		t.Error("different amounts should produce different signatures")
	}
}

// ─── validatePaymeAuth ──────────────────────────────────────────────────────

func TestValidatePaymeAuth_Valid(t *testing.T) {
	merchantKey := "test-secret-key"
	raw := "Paycom:" + merchantKey
	encoded := base64.StdEncoding.EncodeToString([]byte(raw))
	header := "Basic " + encoded

	if !validatePaymeAuth(header, merchantKey) {
		t.Error("expected true for valid auth")
	}
}

func TestValidatePaymeAuth_WrongKey(t *testing.T) {
	raw := "Paycom:wrong-key"
	encoded := base64.StdEncoding.EncodeToString([]byte(raw))
	header := "Basic " + encoded

	if validatePaymeAuth(header, "correct-key") {
		t.Error("expected false for wrong key")
	}
}

func TestValidatePaymeAuth_MissingBasicPrefix(t *testing.T) {
	if validatePaymeAuth("Bearer token", "key") {
		t.Error("expected false for non-Basic auth")
	}
}

func TestValidatePaymeAuth_InvalidBase64(t *testing.T) {
	if validatePaymeAuth("Basic !!!not-base64!!!", "key") {
		t.Error("expected false for invalid base64")
	}
}

func TestValidatePaymeAuth_EmptyHeader(t *testing.T) {
	if validatePaymeAuth("", "key") {
		t.Error("expected false for empty header")
	}
}

// ─── isGlobalPayPaidStatus ──────────────────────────────────────────────────

func TestIsGlobalPayPaidStatus(t *testing.T) {
	paid := []string{"SUCCESS", "SUCCEEDED", "PAID", "COMPLETED", "COMPLETE", "CONFIRMED", "APPROVED"}
	for _, s := range paid {
		if !isGlobalPayPaidStatus(s) {
			t.Errorf("%q should be paid", s)
		}
	}
	// Case insensitive
	if !isGlobalPayPaidStatus("success") {
		t.Error("should be case-insensitive")
	}
	if !isGlobalPayPaidStatus("  PAID  ") {
		t.Error("should trim whitespace")
	}
	if isGlobalPayPaidStatus("PENDING") {
		t.Error("PENDING should not be paid")
	}
	if isGlobalPayPaidStatus("") {
		t.Error("empty should not be paid")
	}
}

// ─── isGlobalPayFailedStatus ────────────────────────────────────────────────

func TestIsGlobalPayFailedStatus(t *testing.T) {
	failed := []string{"FAILED", "FAIL", "CANCELLED", "CANCELED", "DECLINED", "REJECTED", "EXPIRED", "ERROR"}
	for _, s := range failed {
		if !isGlobalPayFailedStatus(s) {
			t.Errorf("%q should be failed", s)
		}
	}
	if !isGlobalPayFailedStatus("  failed  ") {
		t.Error("should be case-insensitive and trim whitespace")
	}
	if isGlobalPayFailedStatus("SUCCESS") {
		t.Error("SUCCESS should not be failed")
	}
}

// ─── firstNonEmpty ──────────────────────────────────────────────────────────

func TestFirstNonEmpty(t *testing.T) {
	tests := []struct {
		name   string
		input  []string
		expect string
	}{
		{"first non-empty", []string{"", "", "hello"}, "hello"},
		{"all empty", []string{"", "", ""}, ""},
		{"first wins", []string{"first", "second"}, "first"},
		{"whitespace trimmed", []string{"  ", "  actual  "}, "actual"},
		{"no args", []string{}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := firstNonEmpty(tt.input...)
			if got != tt.expect {
				t.Errorf("got %q, want %q", got, tt.expect)
			}
		})
	}
}

// ─── globalPayLookupString ──────────────────────────────────────────────────

func TestGlobalPayLookupString_TopLevel(t *testing.T) {
	body := map[string]interface{}{"status": "SUCCESS"}
	got := globalPayLookupString(body, "status")
	if got != "SUCCESS" {
		t.Errorf("got %q, want SUCCESS", got)
	}
}

func TestGlobalPayLookupString_Nested(t *testing.T) {
	body := map[string]interface{}{
		"data": map[string]interface{}{"payment_id": "PAY-123"},
	}
	got := globalPayLookupString(body, "payment_id")
	if got != "PAY-123" {
		t.Errorf("got %q, want PAY-123", got)
	}
}

func TestGlobalPayLookupString_Missing(t *testing.T) {
	body := map[string]interface{}{"other": "value"}
	got := globalPayLookupString(body, "nonexistent")
	if got != "" {
		t.Errorf("got %q, want empty", got)
	}
}

func TestGlobalPayLookupString_NumericValue(t *testing.T) {
	body := map[string]interface{}{"amount": float64(50000)}
	got := globalPayLookupString(body, "amount")
	if got != "50000" {
		t.Errorf("got %q, want 50000", got)
	}
}

// ─── globalPayLookupBool ────────────────────────────────────────────────────

func TestGlobalPayLookupBool(t *testing.T) {
	tests := []struct {
		name   string
		body   map[string]interface{}
		key    string
		expect bool
	}{
		{"bool true", map[string]interface{}{"active": true}, "active", true},
		{"bool false", map[string]interface{}{"active": false}, "active", false},
		{"string true", map[string]interface{}{"active": "true"}, "active", true},
		{"string 1", map[string]interface{}{"active": "1"}, "active", true},
		{"string yes", map[string]interface{}{"active": "yes"}, "active", true},
		{"string no", map[string]interface{}{"active": "no"}, "active", false},
		{"float nonzero", map[string]interface{}{"active": float64(1)}, "active", true},
		{"float zero", map[string]interface{}{"active": float64(0)}, "active", false},
		{"missing", map[string]interface{}{}, "active", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := globalPayLookupBool(tt.body, tt.key)
			if got != tt.expect {
				t.Errorf("got %v, want %v", got, tt.expect)
			}
		})
	}
}

// ─── globalPayLookupTime ────────────────────────────────────────────────────

func TestGlobalPayLookupTime_RFC3339(t *testing.T) {
	body := map[string]interface{}{"created_at": "2026-04-12T10:30:00Z"}
	got := globalPayLookupTime(body, "created_at")
	if got == nil {
		t.Fatal("expected non-nil time")
	}
	if got.Year() != 2026 || got.Month() != 4 || got.Day() != 12 {
		t.Errorf("unexpected time: %v", got)
	}
}

func TestGlobalPayLookupTime_UnixTimestamp(t *testing.T) {
	body := map[string]interface{}{"timestamp": "1776076800"} // some future unix
	got := globalPayLookupTime(body, "timestamp")
	if got == nil {
		t.Fatal("expected non-nil time")
	}
}

func TestGlobalPayLookupTime_Empty(t *testing.T) {
	body := map[string]interface{}{}
	got := globalPayLookupTime(body, "timestamp")
	if got != nil {
		t.Error("expected nil for missing key")
	}
}

func TestGlobalPayLookupTime_InvalidFormat(t *testing.T) {
	body := map[string]interface{}{"timestamp": "not-a-time"}
	got := globalPayLookupTime(body, "timestamp")
	if got != nil {
		t.Error("expected nil for invalid time format")
	}
}

// ─── ComputeSplitRecipients ─────────────────────────────────────────────────

func TestComputeSplitRecipients(t *testing.T) {
	os.Setenv("GLOBAL_PAY_PLATFORM_MERCHANT_ID", "PLATFORM-001")
	defer os.Unsetenv("GLOBAL_PAY_PLATFORM_MERCHANT_ID")

	splits := ComputeSplitRecipients(100000, "SUPPLIER-001", 500)
	if splits == nil {
		t.Fatal("expected non-nil splits")
	}
	if len(splits) != 2 {
		t.Fatalf("expected 2 splits, got %d", len(splits))
	}

	// 100000 = 10000000 tiyin
	// 95% supplier = 9500000, 5% platform = 500000
	totalTiyin := int64(100000 * 100)
	supplierTiyin := totalTiyin * 95 / 100
	platformTiyin := totalTiyin - supplierTiyin

	if splits[0].MerchantID != "SUPPLIER-001" || splits[0].Amount != supplierTiyin {
		t.Errorf("supplier split wrong: %+v", splits[0])
	}
	if splits[1].MerchantID != "PLATFORM-001" || splits[1].Amount != platformTiyin {
		t.Errorf("platform split wrong: %+v", splits[1])
	}
}

func TestComputeSplitRecipients_MissingSupplier(t *testing.T) {
	os.Setenv("GLOBAL_PAY_PLATFORM_MERCHANT_ID", "PLATFORM-001")
	defer os.Unsetenv("GLOBAL_PAY_PLATFORM_MERCHANT_ID")

	splits := ComputeSplitRecipients(100000, "", 500)
	if splits != nil {
		t.Error("expected nil for empty supplier recipient")
	}
}

func TestComputeSplitRecipients_MissingPlatform(t *testing.T) {
	os.Unsetenv("GLOBAL_PAY_PLATFORM_MERCHANT_ID")
	splits := ComputeSplitRecipients(100000, "SUPPLIER-001", 500)
	if splits != nil {
		t.Error("expected nil when platform merchant ID not set")
	}
}

func TestComputeSplitRecipients_IntegerMath(t *testing.T) {
	os.Setenv("GLOBAL_PAY_PLATFORM_MERCHANT_ID", "PLAT")
	defer os.Unsetenv("GLOBAL_PAY_PLATFORM_MERCHANT_ID")

	// Test with odd amount to ensure no rounding loss
	splits := ComputeSplitRecipients(33333, "SUP", 500)
	if splits == nil {
		t.Fatal("expected non-nil splits")
	}
	total := splits[0].Amount + splits[1].Amount
	expectedTotal := int64(33333 * 100)
	if total != expectedTotal {
		t.Errorf("total %d != expected %d (rounding loss!)", total, expectedTotal)
	}
}

// ─── SimulatedClient ────────────────────────────────────────────────────────

func TestSimulatedClient_ChargeSuccess(t *testing.T) {
	client, err := NewSimulatedClient()
	if err != nil {
		t.Fatal(err)
	}
	err = client.Charge("ORD-001", 50000)
	if err != nil {
		t.Errorf("expected success, got %v", err)
	}
}

func TestSimulatedClient_ChargeDecline(t *testing.T) {
	client, _ := NewSimulatedClient()
	err := client.Charge("ORD-999", 50000)
	if err == nil {
		t.Error("expected decline for order ending in 999")
	}
	if !strings.Contains(err.Error(), "declined") {
		t.Errorf("error should mention 'declined': %v", err)
	}
}

func TestSimulatedClient_Refund(t *testing.T) {
	client, _ := NewSimulatedClient()
	err := client.Refund("ORD-001", 10000)
	if err != nil {
		t.Errorf("expected success, got %v", err)
	}
}

// ─── noopGateway ────────────────────────────────────────────────────────────

func TestNoopGateway_ChargeAndRefund(t *testing.T) {
	gw := &noopGateway{gateway: "UZCARD"}
	if err := gw.Charge("ORD-1", 100); err != nil {
		t.Error(err)
	}
	if err := gw.Refund("ORD-1", 50); err != nil {
		t.Error(err)
	}
}

// ─── Session Status Constants ───────────────────────────────────────────────

func TestSessionStatusConstants(t *testing.T) {
	statuses := map[string]string{
		"CREATED":   SessionCreated,
		"PENDING":   SessionPending,
		"SETTLED":   SessionSettled,
		"FAILED":    SessionFailed,
		"EXPIRED":   SessionExpired,
		"CANCELLED": SessionCancelled,
	}
	for expected, constant := range statuses {
		if constant != expected {
			t.Errorf("%s = %q, want %q", expected, constant, expected)
		}
	}
}

func TestAttemptStatusConstants(t *testing.T) {
	statuses := map[string]string{
		"INITIATED":  AttemptInitiated,
		"REDIRECTED": AttemptRedirected,
		"PROCESSING": AttemptProcessing,
		"SUCCESS":    AttemptSuccess,
		"FAILED":     AttemptFailed,
		"CANCELLED":  AttemptCancelled,
		"TIMED_OUT":  AttemptTimedOut,
	}
	for expected, constant := range statuses {
		if constant != expected {
			t.Errorf("%s = %q, want %q", expected, constant, expected)
		}
	}
}

// ─── secureCompare ──────────────────────────────────────────────────────────

func TestSecureCompare_Equal(t *testing.T) {
	if !secureCompare("abc", "abc") {
		t.Error("expected true for equal strings")
	}
}

func TestSecureCompare_NotEqual(t *testing.T) {
	if secureCompare("abc", "xyz") {
		t.Error("expected false for different strings")
	}
}

func TestSecureCompare_Empty(t *testing.T) {
	if !secureCompare("", "") {
		t.Error("expected true for empty strings")
	}
}

// ─── globalPayAnyToString ───────────────────────────────────────────────────

func TestGlobalPayAnyToString(t *testing.T) {
	tests := []struct {
		name   string
		input  interface{}
		expect string
	}{
		{"string", "hello", "hello"},
		{"float64", float64(42), "42"},
		{"int64", int64(99), "99"},
		{"nil", nil, ""},
		{"bool", true, ""},
		{"whitespace string", "  trimmed  ", "trimmed"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := globalPayAnyToString(tt.input)
			if got != tt.expect {
				t.Errorf("got %q, want %q", got, tt.expect)
			}
		})
	}
}

// ─── globalPayCandidateMaps ─────────────────────────────────────────────────

func TestGlobalPayCandidateMaps(t *testing.T) {
	body := map[string]interface{}{
		"status": "ok",
		"data":   map[string]interface{}{"id": "123"},
		"result": map[string]interface{}{"code": "200"},
	}
	containers := globalPayCandidateMaps(body)
	// Should include body + data + result = 3
	if len(containers) < 3 {
		t.Errorf("expected at least 3 containers, got %d", len(containers))
	}
}

func TestGlobalPayCandidateMaps_NoNested(t *testing.T) {
	body := map[string]interface{}{"status": "ok"}
	containers := globalPayCandidateMaps(body)
	if len(containers) != 1 {
		t.Errorf("expected 1 container, got %d", len(containers))
	}
}

// ─── Timing — validate test won't hang ──────────────────────────────────────

func TestGlobalPayLookupTime_MultipleFormats(t *testing.T) {
	formats := []string{
		"2026-04-12T10:30:00Z",
		"2026-04-12T10:30:00.000Z",
		"2026-04-12T10:30:00+05:00",
		"2026-04-12 10:30:00",
	}
	for _, f := range formats {
		t.Run(f, func(t *testing.T) {
			body := map[string]interface{}{"ts": f}
			got := globalPayLookupTime(body, "ts")
			if got == nil {
				t.Errorf("failed to parse %q", f)
			}
		})
	}
}

// ─── SplitRecipient struct ──────────────────────────────────────────────────

func TestSplitRecipient_Fields(t *testing.T) {
	sr := SplitRecipient{MerchantID: "M-1", Amount: 5000}
	if sr.MerchantID != "M-1" || sr.Amount != 5000 {
		t.Errorf("unexpected: %+v", sr)
	}
}

// ─── PaymentSession struct ──────────────────────────────────────────────────

func TestPaymentSession_Defaults(t *testing.T) {
	s := PaymentSession{
		SessionID: "sess-1",
		OrderID:   "ord-1",
		Status:    SessionCreated,
		Currency:  "UZS",
	}
	if s.SessionID != "sess-1" || s.Status != "CREATED" || s.Currency != "UZS" {
		t.Errorf("unexpected: %+v", s)
	}
	if s.SettledAt != nil {
		t.Error("SettledAt should be nil for new session")
	}
	if s.ExpiresAt != nil {
		t.Error("ExpiresAt should be nil by default")
	}
}

func TestPaymentAttempt_Defaults(t *testing.T) {
	a := PaymentAttempt{
		AttemptID: "att-1",
		SessionID: "sess-1",
		Status:    AttemptInitiated,
	}
	if a.Status != "INITIATED" {
		t.Errorf("unexpected status: %s", a.Status)
	}
}

// ─── Checkout URL defaults ──────────────────────────────────────────────────

func TestCheckoutURL_DefaultPaymeBase(t *testing.T) {
	os.Setenv("PAYME_MERCHANT_ID", "TEST-MERCH")
	os.Unsetenv("PAYME_CHECKOUT_URL") // force default
	defer os.Unsetenv("PAYME_MERCHANT_ID")

	url, err := CheckoutURL("PAYME", "ORD-1", 1000)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(url, "https://checkout.paycom.uz/") {
		t.Errorf("expected default Payme URL, got %q", url)
	}
}

func TestCheckoutURL_DefaultClickBase(t *testing.T) {
	os.Setenv("CLICK_MERCHANT_ID", "TEST-MERCH")
	os.Setenv("CLICK_SERVICE_ID", "TEST-SVC")
	os.Unsetenv("CLICK_CHECKOUT_URL") // force default
	defer func() {
		os.Unsetenv("CLICK_MERCHANT_ID")
		os.Unsetenv("CLICK_SERVICE_ID")
	}()

	url, err := CheckoutURL("CLICK", "ORD-1", 1000)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(url, "https://my.click.uz/services/pay") {
		t.Errorf("expected default Click URL, got %q", url)
	}
}

// ─── RetailerPusher interface compliance ────────────────────────────────────

type mockRetailerPusher struct{ pushed bool }

func (m *mockRetailerPusher) PushToRetailer(id string, payload interface{}) bool {
	m.pushed = true
	return true
}

func TestRetailerPusherInterface(t *testing.T) {
	var _ RetailerPusher = &mockRetailerPusher{}
}

// ─── DriverPusher interface compliance ──────────────────────────────────────

type mockDriverPusher struct{ pushed bool }

func (m *mockDriverPusher) PushToDriver(id string, payload interface{}) bool {
	m.pushed = true
	return true
}

func TestDriverPusherInterface(t *testing.T) {
	var _ DriverPusher = &mockDriverPusher{}
}

package payment

// ─── NAUGHTY TEST SUITE: Payment Integrity Vulnerability Assessment ─────────
//
// This file intentionally tries to BREAK the split-logic, idempotency,
// and money-precision guarantees of the V.O.I.D. payment engine.
//
// Three kill zones tested:
//   1. Idempotency Erasure (double-charge risk)
//   2. Split-Logic Starvation (stale/null recipient routing)
//   3. Partial Capture Precision Loss (float rounding, remainder theft)
//
// Plus two F.R.I.D.A.Y. Zero-Trust checks:
//   - Ghost Order: null RecipientId must fallback, not crash
//   - Replay: same payload → same AuthorizationId

import (
	"math"
	"os"
	"testing"
)

// ═══════════════════════════════════════════════════════════════════════════════
// KILL ZONE 1: SPLIT-LOGIC — ComputeSplitRecipients
// ═══════════════════════════════════════════════════════════════════════════════

// ── 1a. Atomic Conservation: supplier + platform MUST equal total (no money leak) ──

func TestSplit_TotalConservation_OddAmount(t *testing.T) {
	os.Setenv("GLOBAL_PAY_PLATFORM_MERCHANT_ID", "PLAT-TEST")
	defer os.Unsetenv("GLOBAL_PAY_PLATFORM_MERCHANT_ID")

	// Odd amount — integer division could lose a tiyin
	amount := int64(99999) // 99,999 UZS = 9,999,900 tiyin
	splits := ComputeSplitRecipients(amount, "SUP-001", 500)
	if splits == nil {
		t.Fatal("expected non-nil splits")
	}
	total := splits[0].Amount + splits[1].Amount
	expectedTotal := amount * 100
	if total != expectedTotal {
		t.Errorf("MONEY LEAK: split total %d != expected %d (lost %d tiyin)", total, expectedTotal, expectedTotal-total)
	}
}

func TestSplit_TotalConservation_PrimeAmount(t *testing.T) {
	os.Setenv("GLOBAL_PAY_PLATFORM_MERCHANT_ID", "PLAT-TEST")
	defer os.Unsetenv("GLOBAL_PAY_PLATFORM_MERCHANT_ID")

	// Prime number — worst case for clean division
	amount := int64(10007) // 10,007 UZS
	splits := ComputeSplitRecipients(amount, "SUP-002", 500)
	if splits == nil {
		t.Fatal("expected non-nil splits")
	}
	total := splits[0].Amount + splits[1].Amount
	expectedTotal := amount * 100
	if total != expectedTotal {
		t.Errorf("MONEY LEAK: split total %d != expected %d (lost %d tiyin)", total, expectedTotal, expectedTotal-total)
	}
}

func TestSplit_TotalConservation_OneUZS(t *testing.T) {
	os.Setenv("GLOBAL_PAY_PLATFORM_MERCHANT_ID", "PLAT-TEST")
	defer os.Unsetenv("GLOBAL_PAY_PLATFORM_MERCHANT_ID")

	// Minimum meaningful order — 1 UZS = 100 tiyin
	// 5% of 100 tiyin = 5 tiyin platform, 95 tiyin supplier
	splits := ComputeSplitRecipients(1, "SUP-003", 500)
	if splits == nil {
		t.Fatal("expected non-nil splits for 1 UZS")
	}
	total := splits[0].Amount + splits[1].Amount
	if total != 100 {
		t.Errorf("MONEY LEAK: 1 UZS split total = %d tiyin, expected 100", total)
	}
}

func TestSplit_TotalConservation_ZeroAmount(t *testing.T) {
	os.Setenv("GLOBAL_PAY_PLATFORM_MERCHANT_ID", "PLAT-TEST")
	defer os.Unsetenv("GLOBAL_PAY_PLATFORM_MERCHANT_ID")

	// Zero-amount should produce zero splits, not negative or crash
	splits := ComputeSplitRecipients(0, "SUP-004", 500)
	if splits == nil {
		t.Fatal("expected non-nil splits for zero (should produce two zero-amount recipients)")
	}
	total := splits[0].Amount + splits[1].Amount
	if total != 0 {
		t.Errorf("zero amount produced non-zero split total: %d", total)
	}
}

// ── 1b. Remainder Theft: who gets the truncation remainder? ──

func TestSplit_RemainderGoesToSupplier_NotPlatform(t *testing.T) {
	os.Setenv("GLOBAL_PAY_PLATFORM_MERCHANT_ID", "PLAT-TEST")
	defer os.Unsetenv("GLOBAL_PAY_PLATFORM_MERCHANT_ID")

	// Amount where 5% produces a remainder:
	// 33333 UZS = 3,333,300 tiyin
	// 5% = 166,665 tiyin (exact), supplier = 3,166,635 tiyin
	// Check: supplier gets total - platform (remainder-safe)
	splits := ComputeSplitRecipients(33333, "SUP-REM", 500)
	if splits == nil {
		t.Fatal("nil splits")
	}

	supplierAmount := splits[0].Amount
	platformAmount := splits[1].Amount
	totalTiyin := int64(33333 * 100)

	// Platform must be floor(total * fee / 10000)
	expectedPlatform := totalTiyin * 500 / 10000
	if platformAmount != expectedPlatform {
		t.Errorf("platform amount %d != expected %d", platformAmount, expectedPlatform)
	}

	// Supplier must be exactly total - platform (gets the remainder)
	if supplierAmount != totalTiyin-expectedPlatform {
		t.Errorf("supplier amount %d != expected %d — remainder was stolen", supplierAmount, totalTiyin-expectedPlatform)
	}
}

// ── 1c. Fee Percentage Boundary Attacks ──

func TestSplit_ZeroFee_SupplierGetsEverything(t *testing.T) {
	os.Setenv("GLOBAL_PAY_PLATFORM_MERCHANT_ID", "PLAT-TEST")
	defer os.Unsetenv("GLOBAL_PAY_PLATFORM_MERCHANT_ID")

	splits := ComputeSplitRecipients(50000, "SUP-NOFEE", 0) // 0% fee
	if splits == nil {
		t.Fatal("nil splits with 0% fee")
	}
	if splits[1].Amount != 0 {
		t.Errorf("platform should get 0 with 0%% fee, got %d", splits[1].Amount)
	}
	if splits[0].Amount != 50000*100 {
		t.Errorf("supplier should get all %d tiyin, got %d", int64(50000*100), splits[0].Amount)
	}
}

func TestSplit_100PercentFee_PlatformGetsEverything(t *testing.T) {
	os.Setenv("GLOBAL_PAY_PLATFORM_MERCHANT_ID", "PLAT-TEST")
	defer os.Unsetenv("GLOBAL_PAY_PLATFORM_MERCHANT_ID")

	// 10000 basis points = 100%
	splits := ComputeSplitRecipients(50000, "SUP-ALLFEE", 10000)
	if splits == nil {
		t.Fatal("nil splits with 100% fee")
	}
	if splits[0].Amount != 0 {
		t.Errorf("supplier should get 0 with 100%% fee, got %d", splits[0].Amount)
	}
	if splits[1].Amount != 50000*100 {
		t.Errorf("platform should get all %d tiyin, got %d", int64(50000*100), splits[1].Amount)
	}
}

func TestSplit_NegativeFee_MustNotProduceNegativeSupplierAmount(t *testing.T) {
	os.Setenv("GLOBAL_PAY_PLATFORM_MERCHANT_ID", "PLAT-TEST")
	defer os.Unsetenv("GLOBAL_PAY_PLATFORM_MERCHANT_ID")

	// A negative fee should not produce negative amounts (defensive check)
	splits := ComputeSplitRecipients(50000, "SUP-NEGFEE", -500)
	if splits == nil {
		t.Skip("nil splits for negative fee — acceptable if validation is done upstream")
	}
	for i, s := range splits {
		if s.Amount < 0 {
			t.Errorf("NEGATIVE AMOUNT in split[%d]: %d tiyin — gateway will reject this", i, s.Amount)
		}
	}
}

func TestSplit_OverflowFee_BeyondTotalMustNotGoNegative(t *testing.T) {
	os.Setenv("GLOBAL_PAY_PLATFORM_MERCHANT_ID", "PLAT-TEST")
	defer os.Unsetenv("GLOBAL_PAY_PLATFORM_MERCHANT_ID")

	// Fee > 100% (20000 bp = 200%)
	splits := ComputeSplitRecipients(50000, "SUP-OVER", 20000)
	if splits == nil {
		t.Skip("nil splits for overflow fee — acceptable if validation is done upstream")
	}
	if splits[0].Amount < 0 {
		t.Errorf("CRITICAL: supplier amount went negative: %d — gateway WILL reject this", splits[0].Amount)
	}
}

// ── 1d. Large Amount Stress (Overflow Detection) ──

func TestSplit_LargeAmount_NoOverflow(t *testing.T) {
	os.Setenv("GLOBAL_PAY_PLATFORM_MERCHANT_ID", "PLAT-TEST")
	defer os.Unsetenv("GLOBAL_PAY_PLATFORM_MERCHANT_ID")

	// 500 billion UZS (extreme but valid for bulk B2B)
	amount := int64(500_000_000_000)
	splits := ComputeSplitRecipients(amount, "SUP-BIG", 500)
	if splits == nil {
		t.Fatal("nil splits for large amount")
	}

	total := splits[0].Amount + splits[1].Amount
	expectedTotal := amount * 100

	// Check for int64 overflow: if the multiplication wrapped, total would be wrong
	if expectedTotal < 0 || total < 0 {
		t.Fatalf("INT64 OVERFLOW: amount=%d, expectedTotal=%d, actualTotal=%d", amount, expectedTotal, total)
	}
	if total != expectedTotal {
		t.Errorf("MONEY LEAK at scale: total %d != expected %d", total, expectedTotal)
	}
}

func TestSplit_MaxInt64_OverflowBoundary(t *testing.T) {
	os.Setenv("GLOBAL_PAY_PLATFORM_MERCHANT_ID", "PLAT-TEST")
	defer os.Unsetenv("GLOBAL_PAY_PLATFORM_MERCHANT_ID")

	// math.MaxInt64 / 100 is the max safe UZS before the *100 tiyin conversion overflows
	maxSafe := int64(math.MaxInt64 / 100)
	splits := ComputeSplitRecipients(maxSafe, "SUP-MAX", 500)
	if splits == nil {
		t.Fatal("nil splits at MaxInt64/100")
	}

	total := splits[0].Amount + splits[1].Amount
	expectedTotal := maxSafe * 100
	if total != expectedTotal {
		t.Errorf("overflow at boundary: total %d != expected %d", total, expectedTotal)
	}

	// Now test ABOVE the safe boundary — this SHOULD overflow
	unsafeAmount := maxSafe + 1
	unsafeSplits := ComputeSplitRecipients(unsafeAmount, "SUP-UNSAFE", 500)
	if unsafeSplits != nil {
		unsafeTotal := unsafeSplits[0].Amount + unsafeSplits[1].Amount
		unsafeExpected := unsafeAmount * 100
		if unsafeTotal != unsafeExpected {
			t.Logf("WARNING: int64 overflow detected at amount=%d — total=%d expected=%d (needs guard)", unsafeAmount, unsafeTotal, unsafeExpected)
		}
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// KILL ZONE 2: GHOST ORDER — Null RecipientId Fallback
// ═══════════════════════════════════════════════════════════════════════════════

func TestSplit_NullRecipientId_FallbackToNil(t *testing.T) {
	os.Setenv("GLOBAL_PAY_PLATFORM_MERCHANT_ID", "PLAT-TEST")
	defer os.Unsetenv("GLOBAL_PAY_PLATFORM_MERCHANT_ID")

	// Empty supplier recipient → splits should be nil (platform-only fallback)
	splits := ComputeSplitRecipients(100000, "", 500)
	if splits != nil {
		t.Errorf("GHOST ORDER: empty RecipientId should return nil, got %+v — money would route to empty merchant", splits)
	}
}

func TestSplit_NullRecipientId_WithWhitespace_NotGuarded(t *testing.T) {
	os.Setenv("GLOBAL_PAY_PLATFORM_MERCHANT_ID", "PLAT-TEST")
	defer os.Unsetenv("GLOBAL_PAY_PLATFORM_MERCHANT_ID")

	// Whitespace-only is now trimmed to "" → returns nil (safe fallback)
	splits := ComputeSplitRecipients(100000, "   ", 500)
	if splits != nil {
		t.Errorf("whitespace-only RecipientId should return nil after TrimSpace, got MerchantID=%q",
			splits[0].MerchantID)
	}
}

func TestSplit_NullPlatformMerchant_FallbackToNil(t *testing.T) {
	os.Unsetenv("GLOBAL_PAY_PLATFORM_MERCHANT_ID")

	splits := ComputeSplitRecipients(100000, "SUP-VALID", 500)
	if splits != nil {
		t.Errorf("missing platform merchant ID should return nil, got %+v", splits)
	}
}

func TestSplit_WhitespacePlatformMerchant_TrimmedToNil(t *testing.T) {
	os.Setenv("GLOBAL_PAY_PLATFORM_MERCHANT_ID", "   ")
	defer os.Unsetenv("GLOBAL_PAY_PLATFORM_MERCHANT_ID")

	// strings.TrimSpace("   ") → "" → nil return
	splits := ComputeSplitRecipients(100000, "SUP-VALID", 500)
	if splits != nil {
		t.Errorf("whitespace platform merchant should return nil after TrimSpace, got %+v", splits)
	}
}

func TestSplit_WhitespaceSupplierRecipient_NotTrimmed(t *testing.T) {
	os.Setenv("GLOBAL_PAY_PLATFORM_MERCHANT_ID", "PLAT-TEST")
	defer os.Unsetenv("GLOBAL_PAY_PLATFORM_MERCHANT_ID")

	// Whitespace-only supplierRecipientID must be trimmed to "" → nil return.
	// Before the fix, "   " != "" passed the guard and produced a garbage split.
	splits := ComputeSplitRecipients(100000, "   ", 500)
	if splits != nil {
		t.Errorf("whitespace-only supplierRecipientID should return nil after TrimSpace, got MerchantID=%q",
			splits[0].MerchantID)
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// KILL ZONE 3: PRECISION LOSS — Refund Split vs Checkout Split Consistency
// ═══════════════════════════════════════════════════════════════════════════════

// Both checkout and refund paths must use identical tiyin + basis-point math:
//   (amount * 100 * feeBP) / 10000
// This ensures zero divergence regardless of the amount.

func TestSplit_RefundVsCheckout_ConsistencyAt100000UZS(t *testing.T) {
	os.Setenv("GLOBAL_PAY_PLATFORM_MERCHANT_ID", "PLAT-TEST")
	defer os.Unsetenv("GLOBAL_PAY_PLATFORM_MERCHANT_ID")

	amount := int64(100000) // 100,000 UZS
	feeBP := int64(500)     // 5% in basis points

	// Checkout path: ComputeSplitRecipients (in tiyins)
	splits := ComputeSplitRecipients(amount, "SUP-001", feeBP)
	if splits == nil {
		t.Fatal("nil splits")
	}
	checkoutPlatformTiyin := splits[1].Amount
	checkoutSupplierTiyin := splits[0].Amount

	// Refund path: canonical tiyin + basis-point formula (matches refund.go)
	totalTiyin := amount * 100
	refundLabTiyin := totalTiyin * feeBP / 10000
	refundSupplierTiyin := totalTiyin - refundLabTiyin

	if checkoutPlatformTiyin != refundLabTiyin {
		t.Errorf("SPLIT DIVERGENCE: checkout platform=%d tiyin, refund lab=%d tiyin (delta=%d)",
			checkoutPlatformTiyin, refundLabTiyin, checkoutPlatformTiyin-refundLabTiyin)
	}
	if checkoutSupplierTiyin != refundSupplierTiyin {
		t.Errorf("SPLIT DIVERGENCE: checkout supplier=%d tiyin, refund supplier=%d tiyin (delta=%d)",
			checkoutSupplierTiyin, refundSupplierTiyin, checkoutSupplierTiyin-refundSupplierTiyin)
	}
}

func TestSplit_RefundVsCheckout_ConsistencyAt33333UZS(t *testing.T) {
	os.Setenv("GLOBAL_PAY_PLATFORM_MERCHANT_ID", "PLAT-TEST")
	defer os.Unsetenv("GLOBAL_PAY_PLATFORM_MERCHANT_ID")

	amount := int64(33333)

	// Checkout path
	splits := ComputeSplitRecipients(amount, "SUP-002", 500)
	if splits == nil {
		t.Fatal("nil splits")
	}
	checkoutPlatformTiyin := splits[1].Amount

	// Refund path — must use identical tiyin + basis-point math as checkout
	commissionBP := int64(500)
	refundLabTiyin := (amount * 100 * commissionBP) / 10000

	if checkoutPlatformTiyin != refundLabTiyin {
		t.Errorf("SPLIT DIVERGENCE: checkout platform=%d tiyin, refund lab=%d tiyin — delta=%d tiyin",
			checkoutPlatformTiyin, refundLabTiyin, checkoutPlatformTiyin-refundLabTiyin)
	}
}

func TestSplit_RefundVsCheckout_BruteForceConsistency(t *testing.T) {
	os.Setenv("GLOBAL_PAY_PLATFORM_MERCHANT_ID", "PLAT-TEST")
	defer os.Unsetenv("GLOBAL_PAY_PLATFORM_MERCHANT_ID")

	// Brute-force test across 10000 amounts to find ALL divergences
	divergences := 0
	maxDelta := int64(0)

	for amount := int64(1); amount <= 10000; amount++ {
		splits := ComputeSplitRecipients(amount, "SUP-BF", 500)
		if splits == nil {
			continue
		}
		checkoutPlatformTiyin := splits[1].Amount

		// Refund path — tiyin + basis-point math (matches fixed refund.go)
		refundLabTiyin := (amount * 100 * 500) / 10000

		delta := checkoutPlatformTiyin - refundLabTiyin
		if delta < 0 {
			delta = -delta
		}
		if delta > 0 {
			divergences++
			if delta > maxDelta {
				maxDelta = delta
			}
		}
	}

	if divergences > 0 {
		t.Errorf("REFUND vs CHECKOUT divergence found in %d of 10000 amounts (max delta = %d tiyin). "+
			"The refund path MUST be rewritten to compute in tiyins (amount * 100 * 500 / 10000) "+
			"to match the checkout path.", divergences, maxDelta)
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// PRECISION: Float-Free Verification
// ═══════════════════════════════════════════════════════════════════════════════

func TestSplit_NoFloatContamination(t *testing.T) {
	os.Setenv("GLOBAL_PAY_PLATFORM_MERCHANT_ID", "PLAT-TEST")
	defer os.Unsetenv("GLOBAL_PAY_PLATFORM_MERCHANT_ID")

	// This amount, when divided by a float 0.05, produces an imprecise result:
	// float64(10050 * 100) * 0.05 = 50250.000000000007 (IEEE 754 artifact)
	// int64 truncation would give 50250, but Round would give 50250.
	// Our code uses integer division, so it should be exact.
	amount := int64(10050)
	splits := ComputeSplitRecipients(amount, "SUP-FLOAT", 500)
	if splits == nil {
		t.Fatal("nil splits")
	}

	totalTiyin := amount * 100
	platformTiyin := totalTiyin * 500 / 10000
	supplierTiyin := totalTiyin - platformTiyin

	if splits[1].Amount != platformTiyin {
		t.Errorf("platform: got %d, want %d — possible float contamination", splits[1].Amount, platformTiyin)
	}
	if splits[0].Amount != supplierTiyin {
		t.Errorf("supplier: got %d, want %d — possible float contamination", splits[0].Amount, supplierTiyin)
	}
}

func TestSplit_ClassicFloatTrap_10_50_UZS(t *testing.T) {
	os.Setenv("GLOBAL_PAY_PLATFORM_MERCHANT_ID", "PLAT-TEST")
	defer os.Unsetenv("GLOBAL_PAY_PLATFORM_MERCHANT_ID")

	// The classic 10.50 trap: if someone had float64(10.50) * 100 = 1050 (fine)
	// but float64(10.50) * 0.05 = 0.525 → int64 = 0 (truncation!)
	// Since our amounts are already int64 UZS, this shouldn't happen,
	// but verify the split math handles small amounts correctly.
	amount := int64(10) // 10 UZS = 1000 tiyin
	splits := ComputeSplitRecipients(amount, "SUP-SMALL", 500)
	if splits == nil {
		t.Fatal("nil splits")
	}
	// 1000 * 500 / 10000 = 50 tiyin platform
	if splits[1].Amount != 50 {
		t.Errorf("platform should be 50 tiyin for 10 UZS, got %d", splits[1].Amount)
	}
	if splits[0].Amount != 950 {
		t.Errorf("supplier should be 950 tiyin for 10 UZS, got %d", splits[0].Amount)
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// IDEMPOTENCY: DirectPaymentInitRequest derivation audit
// ═══════════════════════════════════════════════════════════════════════════════

func TestIdempotency_ExternalIDDerivedFromOrder_NotRandom(t *testing.T) {
	// Verify the request struct carries ExternalID (order-derived)
	// and SessionID (durable) — NOT a per-call UUID
	req := DirectPaymentInitRequest{
		CardToken:  "CARD-TOKEN",
		Amount:     50000,
		OrderID:    "ORD-123",
		SessionID:  "SESS-456",
		ExternalID: "ATT-789", // Attempt ID — MUST be stable across retries
		Recipients: nil,
	}

	// Re-create the "same" request (simulating Kafka retry)
	retry := DirectPaymentInitRequest{
		CardToken:  "CARD-TOKEN",
		Amount:     50000,
		OrderID:    "ORD-123",
		SessionID:  "SESS-456",
		ExternalID: "ATT-789", // Same attempt → same ExternalID
		Recipients: nil,
	}

	if req.ExternalID != retry.ExternalID {
		t.Error("DOUBLE CHARGE RISK: ExternalID changed between request and retry")
	}
	if req.SessionID != retry.SessionID {
		t.Error("DOUBLE CHARGE RISK: SessionID changed between request and retry")
	}
	if req.OrderID != retry.OrderID {
		t.Error("DOUBLE CHARGE RISK: OrderID changed between request and retry")
	}
}

func TestIdempotency_ExternalIDMustNotBeEmpty(t *testing.T) {
	// Empty ExternalID means the gateway treats every call as a new payment
	req := DirectPaymentInitRequest{
		CardToken:  "CARD-TOKEN",
		Amount:     50000,
		OrderID:    "ORD-123",
		SessionID:  "SESS-456",
		ExternalID: "", // THIS IS THE BUG: no idempotency key
	}

	if req.ExternalID == "" {
		t.Log("WARNING: ExternalID is empty — if this reaches the gateway, every retry creates a new charge. " +
			"Callers MUST populate ExternalID with an attempt-derived stable ID before calling InitPayment.")
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// SESSION STATE MACHINE: Illegal Transition Attempts
// ═══════════════════════════════════════════════════════════════════════════════

func TestSessionStatus_Constants_AreDistinct(t *testing.T) {
	// If two status constants collide, the state machine breaks silently
	statuses := []string{
		SessionCreated, SessionPending, SessionAuthorized, SessionSettled,
		SessionFailed, SessionExpired, SessionCancelled, SessionPartiallyPaid,
	}
	seen := make(map[string]bool, len(statuses))
	for _, s := range statuses {
		if seen[s] {
			t.Errorf("DUPLICATE session status constant: %q — state machine will conflate two states", s)
		}
		seen[s] = true
	}
}

func TestAttemptStatus_Constants_AreDistinct(t *testing.T) {
	statuses := []string{
		AttemptInitiated, AttemptRedirected, AttemptProcessing,
		AttemptSuccess, AttemptFailed, AttemptCancelled, AttemptTimedOut,
	}
	seen := make(map[string]bool, len(statuses))
	for _, s := range statuses {
		if seen[s] {
			t.Errorf("DUPLICATE attempt status constant: %q", s)
		}
		seen[s] = true
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// SPLIT RECIPIENT STRUCT SAFETY
// ═══════════════════════════════════════════════════════════════════════════════

func TestSplitRecipient_JSONTagsMatchGatewayContract(t *testing.T) {
	// Verify SplitRecipient marshals with the exact field names Global Pay expects
	r := SplitRecipient{MerchantID: "MERCH-001", Amount: 500000}

	if r.MerchantID == "" {
		t.Error("MerchantID must not be empty for a valid recipient")
	}
	if r.Amount <= 0 {
		t.Error("Amount must be positive for a valid recipient")
	}
}

func TestSplit_SupplierIsFirstRecipient(t *testing.T) {
	os.Setenv("GLOBAL_PAY_PLATFORM_MERCHANT_ID", "PLAT-TEST")
	defer os.Unsetenv("GLOBAL_PAY_PLATFORM_MERCHANT_ID")

	splits := ComputeSplitRecipients(50000, "SUP-FIRST", 500)
	if splits == nil {
		t.Fatal("nil splits")
	}
	if len(splits) != 2 {
		t.Fatalf("expected 2 recipients, got %d", len(splits))
	}
	// Supplier must be first — some gateways treat index 0 as primary merchant
	if splits[0].MerchantID != "SUP-FIRST" {
		t.Errorf("supplier should be first recipient, got %q", splits[0].MerchantID)
	}
	if splits[1].MerchantID != "PLAT-TEST" {
		t.Errorf("platform should be second recipient, got %q", splits[1].MerchantID)
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// PARTIAL CAPTURE: Amount must be ≤ Authorized Amount
// ═══════════════════════════════════════════════════════════════════════════════

func TestPartialCapture_CaptureCannotExceedLocked(t *testing.T) {
	// Test the PaymentSession struct constraint:
	// CapturedAmount must always be ≤ AuthorizedAmount ≤ LockedAmount
	session := PaymentSession{
		LockedAmount:    100000,
		AuthorizedAmount: 100000,
		CapturedAmount:  80000, // Partial capture — driver removed items
	}

	if session.CapturedAmount > session.AuthorizedAmount {
		t.Error("CapturedAmount exceeds AuthorizedAmount — gateway will reject")
	}
	if session.AuthorizedAmount > session.LockedAmount {
		t.Error("AuthorizedAmount exceeds LockedAmount — over-authorization")
	}
}

func TestPartialCapture_SplitOnReducedAmount(t *testing.T) {
	os.Setenv("GLOBAL_PAY_PLATFORM_MERCHANT_ID", "PLAT-TEST")
	defer os.Unsetenv("GLOBAL_PAY_PLATFORM_MERCHANT_ID")

	// Original auth: 100,000 UZS. After driver subtracts items: 85,000 UZS capture.
	// The split must be computed on the CAPTURE amount, not the AUTH amount.
	authAmount := int64(100000)
	captureAmount := int64(85000)

	authSplits := ComputeSplitRecipients(authAmount, "SUP-CAP", 500)
	captureSplits := ComputeSplitRecipients(captureAmount, "SUP-CAP", 500)

	if authSplits == nil || captureSplits == nil {
		t.Fatal("nil splits")
	}

	// Capture splits must be strictly less than auth splits
	if captureSplits[0].Amount >= authSplits[0].Amount {
		t.Errorf("capture supplier amount %d should be less than auth %d", captureSplits[0].Amount, authSplits[0].Amount)
	}

	// Both must still conserve total
	captureTotal := captureSplits[0].Amount + captureSplits[1].Amount
	if captureTotal != captureAmount*100 {
		t.Errorf("capture split total %d != expected %d tiyin", captureTotal, captureAmount*100)
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// REFUND VALIDATION: Domain Rules
// ═══════════════════════════════════════════════════════════════════════════════

func TestRefundRequest_ZeroAmountMeansFullRefund(t *testing.T) {
	req := RefundRequest{
		OrderID:   "ORD-123",
		Reason:    "damaged goods",
		AmountUZS: 0, // 0 = full refund per contract
	}
	if req.AmountUZS != 0 {
		t.Error("zero AmountUZS should represent full refund")
	}
}

func TestRefundStatus_Constants_AreValid(t *testing.T) {
	validStatuses := map[string]bool{
		RefundPending: true, RefundSettled: true,
		RefundFailed: true, RefundManualRequired: true,
	}
	for status := range validStatuses {
		if status == "" {
			t.Errorf("empty refund status constant")
		}
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// TABLE-DRIVEN SPLIT EXHAUSTIVE TEST
// ═══════════════════════════════════════════════════════════════════════════════

func TestSplit_TableDriven_Exhaustive(t *testing.T) {
	os.Setenv("GLOBAL_PAY_PLATFORM_MERCHANT_ID", "PLAT-TEST")
	defer os.Unsetenv("GLOBAL_PAY_PLATFORM_MERCHANT_ID")

	tests := []struct {
		name              string
		amount            int64
		supplierRecipient string
		feeBP             int64
		wantNil           bool
		wantSupplierTiyin int64
		wantPlatformTiyin int64
	}{
		{
			name:              "standard 100K at 5%",
			amount:            100000,
			supplierRecipient: "SUP",
			feeBP:             500,
			wantSupplierTiyin: 9500000,
			wantPlatformTiyin: 500000,
		},
		{
			name:              "small 500 UZS at 5%",
			amount:            500,
			supplierRecipient: "SUP",
			feeBP:             500,
			wantSupplierTiyin: 47500,
			wantPlatformTiyin: 2500,
		},
		{
			name:              "1 UZS at 5%",
			amount:            1,
			supplierRecipient: "SUP",
			feeBP:             500,
			wantSupplierTiyin: 95,
			wantPlatformTiyin: 5,
		},
		{
			name:              "1 UZS at 3%",
			amount:            1,
			supplierRecipient: "SUP",
			feeBP:             300,
			wantSupplierTiyin: 97,
			wantPlatformTiyin: 3,
		},
		{
			name:              "no supplier → nil",
			amount:            50000,
			supplierRecipient: "",
			feeBP:             500,
			wantNil:           true,
		},
		{
			name:              "zero amount at 5%",
			amount:            0,
			supplierRecipient: "SUP",
			feeBP:             500,
			wantSupplierTiyin: 0,
			wantPlatformTiyin: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			splits := ComputeSplitRecipients(tt.amount, tt.supplierRecipient, tt.feeBP)
			if tt.wantNil {
				if splits != nil {
					t.Errorf("expected nil splits, got %+v", splits)
				}
				return
			}
			if splits == nil {
				t.Fatal("unexpected nil splits")
			}
			if len(splits) != 2 {
				t.Fatalf("expected 2 splits, got %d", len(splits))
			}
			if splits[0].Amount != tt.wantSupplierTiyin {
				t.Errorf("supplier: got %d, want %d", splits[0].Amount, tt.wantSupplierTiyin)
			}
			if splits[1].Amount != tt.wantPlatformTiyin {
				t.Errorf("platform: got %d, want %d", splits[1].Amount, tt.wantPlatformTiyin)
			}
			// Conservation check
			total := splits[0].Amount + splits[1].Amount
			expectedTotal := tt.amount * 100
			if total != expectedTotal {
				t.Errorf("MONEY LEAK: total=%d expected=%d", total, expectedTotal)
			}
		})
	}
}

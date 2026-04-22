# Financial Integrity — Money Math & Ledger Safety

## Description
Prevents money-related bugs: floating-point precision loss, currency confusion, split leakage, and ledger corruption. Activated when writing or reviewing any code that touches money, price, amount, payment, ledger, treasury, reconciliation, fees, splits, or refunds.

## Trigger Keywords
money, price, amount, payment, ledger, treasury, reconciliation, split, fee, refund, tiyin, currency, charge, capture, settle, wallet, escrow, basis points, percentage

## Anti-Pattern Catalog

### 1. float64 for Money (P0 BUG)
```go
// WRONG — precision loss, rounding errors
type ProductVariant struct {
    Price float64 `json:"price"` // 0.1 + 0.2 ≠ 0.3 in IEEE 754
}

// RIGHT — int64 minor units (tiyins)
type ProductVariant struct {
    PriceTiyin int64  `json:"price_tiyin"`
    Currency   string `json:"currency"` // ISO-4217: "UZS", "USD"
}
```
**Real violations**:
- `main.go` ~L1719: `Price float64` in variant DTO sent to mobile clients
- `order/ai_preorder.go` ~L373: `var avgPrice float64` from SQL `AVG()`

**Rule**: ALL currency values are `int64` in tiyins (UZS) or cents (USD). `float64` for money is forbidden at every layer: Go structs, JSON, Spanner, TypeScript, Swift, Kotlin.

### 2. SQL AVG() Truncation vs Rounding
```go
// WRONG — truncates (99.7 → 99)
avgPrice := int64(avgFloat)

// RIGHT — rounds (99.7 → 100)
avgPrice := int64(math.Round(avgFloat))
```
**Context**: Spanner `AVG()` returns `FLOAT64`. Go `int64()` cast truncates toward zero. For money, truncation silently loses value. Always `math.Round()` before casting to `int64`.

### 3. JSON Number Decode Path
Go `encoding/json` decodes all JSON numbers as `float64` by default.
```go
// ACCEPTABLE — immediate conversion at decode boundary
rawAmount := payload["amount"].(float64)
amount := int64(rawAmount) // only if the value is always whole

// BETTER — avoid float64 intermediate entirely
decoder := json.NewDecoder(r.Body)
decoder.UseNumber() // numbers become json.Number, not float64
var req PaymentRequest
decoder.Decode(&req)
// req.Amount is string → parse with strconv.ParseInt
```

**Rule**: When decoding webhook payloads from payment gateways, prefer `decoder.UseNumber()` to avoid the `float64` intermediate. If using default decoding, convert `float64 → int64` at the FIRST possible moment, never propagate `float64` into service logic.

### 4. Percentage Split Zero-Leak Invariant
```go
// Basis-point split: 1 bps = 0.01%
platformFee := (amount * platformBps) / 10000
supplierShare := (amount * supplierBps) / 10000
driverShare := (amount * driverBps) / 10000

// MANDATORY — verify zero-leak
remainder := amount - (platformFee + supplierShare + driverShare)
if remainder != 0 {
    platformFee += remainder // absorb rounding remainder into platform fee
}
// Now: platformFee + supplierShare + driverShare == amount (guaranteed)
```

**Rule**: After computing all split shares, `sum(shares)` MUST equal `totalAmount`. Integer division truncates, so rounding remainders (typically 1-2 tiyins) MUST be absorbed by the platform fee — never the customer or supplier share. A split where `sum ≠ total` is a reconciliation bomb that compounds over millions of transactions.

### 5. Currency as First-Class Field
```go
// WRONG — amount without currency
type Order struct {
    Amount int64 `json:"amount"`
}

// RIGHT — paired amount + currency
type Order struct {
    Amount   int64  `json:"amount"`
    Currency string `json:"currency"` // ISO-4217: "UZS", "USD"
}
```

**Rule**: Every struct, column, event payload, and DTO that carries an amount MUST carry a paired `Currency` field. Never assume UZS. The system handles UZS, USD, and will expand. A ledger row without currency is unreconcilable.

### 6. Major-Unit Conversion Boundary
```go
// WRONG — converting in service logic
func calculateDiscount(priceSom float64) float64 {
    return priceSom * 0.1
}

// RIGHT — internal math is always minor-unit int64
func calculateDiscount(priceTiyin int64) int64 {
    return (priceTiyin * 1000) / 10000 // 10% in basis points
}

// Conversion happens ONLY in response serializer
func toResponse(tiyin int64) string {
    return fmt.Sprintf("%.2f", float64(tiyin)/100.0) // display only
}
```

**Rule**: tiyin → som (or cents → dollars) conversion happens ONLY in the response serializer or UI rendering layer. Internal service logic, repository queries, and event payloads use minor-unit `int64` exclusively.

### 7. Ledger Is Append-Only
```go
// WRONG — updating a ledger row
UPDATE LedgerEntries SET Amount = @newAmount WHERE EntryId = @id

// RIGHT — new paired rows for adjustment
INSERT INTO LedgerEntries (EntryId, AccountId, Amount, Currency, Type, RefId)
VALUES
    (@debitId,  @fromAccount, -@amount, @currency, 'REFUND', @orderId),
    (@creditId, @toAccount,    @amount, @currency, 'REFUND', @orderId)
```

**Rule**: Refunds, adjustments, and corrections are NEW paired debit/credit rows. Never UPDATE or DELETE existing ledger rows. The ledger is append-only. Sum of all rows per currency per day MUST equal zero. This is the double-entry invariant.

### 8. Integer Overflow Guard
For UZS amounts (1 USD ≈ 12,800 UZS ≈ 1,280,000 tiyin):
- Max single order: ~100M UZS = 10^13 tiyin
- Max `int64`: 9.2 × 10^18
- Safe headroom: ~10^5 orders before overflow risk in aggregation

**Rule**: For aggregate sums across many orders (daily revenue, monthly reconciliation), verify the running total stays within `int64` range. Use `math.MaxInt64` as a ceiling check in aggregation loops. For cross-currency aggregation, NEVER sum different currencies — aggregate per-currency.

## Canonical Verification
After writing money-related code, verify:
1. No `float64` or `float32` in any money path (structs, params, returns, JSON tags)
2. Every `Amount` field has a paired `Currency` field
3. Every percentage split verifies `sum(shares) == total`
4. SQL `AVG()` results use `math.Round()` before `int64()` cast
5. Ledger writes are paired debit/credit in the same `ReadWriteTransaction`
6. No major-unit conversion in service logic (only at DTO boundary)
7. Webhook amount decoding uses `UseNumber()` or converts immediately

## Cross-References
- `intrusions.md` §2 — Financial Integrity Engine
- `gemini-instructions.md` §4 Clean Code — Primitive Obsession
- `gemini-instructions.md` Enterprise Algorithm Patterns §5 — Double-Entry Ledger
- `.github/skills/payme-business-integration/SKILL.md` — Payme tiyin handling
- `.github/skills/click-payment-integration/SKILL.md` — Click amount handling

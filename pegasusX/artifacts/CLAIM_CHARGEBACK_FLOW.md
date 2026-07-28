# Post-delivery claims & supplier chargebacks

## Timing (what “48 hours” means)

| Moment | When |
|--------|------|
| **Delivery + handshake** | Driver arrives, QR/cash/card settled, fiscal hard-gate succeeds |
| **Order → `COMPLETED`** | **Immediately** after that success path — **not** after two days |
| **Claim / return window** | **`COMPLETED` + 48h** (env `CLAIM_WINDOW_HOURS`) |
| **Chargeback** | When admin **approves** the claim (not at file time) |

So: deliver Monday 10:00 → complete Monday ~10:05 → retailer can file damage/missing claim until Wednesday 10:05.  
The order is **not** left incomplete for two days.

## End-to-end money flow (marketplace style)

```
1. Retailer files POST /v1/orders/{orderID}/claims
   • Order must be COMPLETED + within claim window + owned by retailer
   • Body: claim_type, line_items[{sku, quantity}], photo evidences for damage
   • System prices lines from ORDER unit prices (never client prices)
   • Claim OPEN, amount_minor = Σ(qty × unit_price_minor)

2. Ops / supplier admin reviews evidence
   • POST /v1/claims/{claimID}/approve  (or /reject)

3. On APPROVE (automatic settlement):
   a. LEDGER: PaymentLedgerEntries CHARGEBACK_RECORDED
      → supplier settlement authority reduced by amount_minor
      (same pattern big marketplaces use for seller clawbacks)
   b. If original payment was GLOBAL_PAY card session:
      → partial refund via backoffice perform (action GLOBAL_PAY_REFUND_ACTION, default RF)
      → retailer card gets money back (full or partial)
   c. If CASH / credit leave-behind:
      → LEDGER_ONLY (no card refund; retailer never paid card)
   d. Claim → RESOLVED + CLAIM_RESOLVED event on logistics.exceptions.v1 + main

4. Reverse logistics (warehouse receive of damaged goods) is parallel:
   REVERSE_LOGISTICS_REQUIRED was already emitted at file time when photo required.
```

### How the system knows order / products / amount

| Field | Source of truth |
|-------|-----------------|
| `order_id` | Path + claim row |
| `supplier_id` / `retailer_id` | Order aggregate |
| SKUs + qty claimed | Claim `line_items` |
| Unit price | **Order** unit prices (weighted avg if mixed prices for same SKU) |
| Line amount | `qty × unit_price` with **int64 overflow checks** |
| Claim total | Sum of lines, capped at `OriginalTotalMinor` (else `TotalMinor`) |
| Multi-claim | Prior OPEN/UNDER_REVIEW/APPROVED/RESOLVED claims **reserve qty** so Σ claimed ≤ ordered |
| Payment session / gateway | `PaymentSessions` by `order_id` |
| Provider payment id | Session id / provider reference for GP perform |

## Global Pay — is refund possible?

| Capability | Status in our stack |
|------------|---------------------|
| Checkout token + capture (`perform` action `CP`) | Implemented |
| Status check | Implemented |
| **Partial / full refund** | **Supported commercially** by Global Pay (marketing: full & partial refunds). Code path: `ExecutionActionRefund` → `perform` with `GLOBAL_PAY_REFUND_ACTION` (default `RF`) |
| Confirm action code | **Must confirm with Global Pay merchant support** — action string may be `RF` / `RP` / other on staging |
| Merchant credentials | Required for live refunds (`GLOBAL_PAY_USERNAME` / password) |

**Important:** Global Pay refund returns money to the **cardholder (retailer)**.  
Supplier clawback is **our ledger** (`CHARGEBACK_RECORDED`), independent of GP — that is the “big company” model (Amazon/eBay style: customer refund + seller debit).

Cash COD orders: **no GP refund**; only supplier ledger debit (and optional store credit later).

## APIs

```http
POST /v1/orders/{orderID}/claims
Authorization: Bearer <retailer>
{
  "claim_type": "CONCEALED_DAMAGE",
  "line_items": [{"sku": "SKU-1", "quantity": 2, "reason": "DAMAGED"}],
  "evidences": [{"evidence_type": "PHOTO", "uri": "https://..."}]
}
→ 201 { claim_id, amount_minor, line_items: [{unit_price_minor, amount_minor}], status: "OPEN" }

POST /v1/claims/{claimID}/approve
Authorization: Bearer <admin>
{ "resolution_note": "photo verified" }
→ 200 { claim: {status: "RESOLVED", ...}, settlement: { chargeback_id, amount_minor, mode, gateway_refunded } }

POST /v1/claims/{claimID}/reject
{ "resolution_note": "not damaged" }
```

Settlement `mode` values:

- `LEDGER_ONLY` — cash/credit or skip refund  
- `LEDGER_AND_GATEWAY_REFUND` — GP partial refund succeeded  
- `LEDGER_ONLY_GATEWAY_REFUND_FAILED` — ledger ok; ops must refund in GP portal  
- `IDEMPOTENT_REPLAY` — approve called again on already-resolved claim (no double charge)  

**Idempotency:** chargeback id is deterministic (`chargeback_<claim_id>`); ledger uses InsertOrUpdate on that key.

## Env

| Env | Default | Meaning |
|-----|---------|---------|
| `CLAIM_WINDOW_HOURS` | `48` | Hours after COMPLETED to file |
| `GLOBAL_PAY_REFUND_ACTION` | `RF` | Backoffice perform action for refund |

## Not yet (future)

- Auto-approve under threshold without human  
- Store-credit path for cash retailers  
- Netting chargebacks into supplier payout batch UI  
- Confirmed GP action codes in staging once merchant password is live  

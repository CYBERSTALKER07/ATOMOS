# Credit Ecosystem Behavior

Trade credit for Pegaus B2B wholesale: irreversible enablement, Net terms, reserved exposure, AR open items, aging/dunning, and role parity.

## What it is

- **Credit relationship** between a supplier and a selected retailer (not consumer BNPL).
- **Net terms** (e.g. Net 7–30) resolve due date at credit leave.
- **AR open items** (`ArInvoices`) track principal/balance/aging after credit leave.
- **Freeze / CREDIT_HOLD** is temporary; it does not disable the relationship.

## Who owns what

| Actor | Owns |
|-------|------|
| Supplier (finance) | Program enable, global defaults, per-retailer enable/terms, holds, collections desk |
| Warehouse finance | Same supplier-scoped APIs (parity UI; not a second ledger) |
| Retailer | Visibility of partners + open invoices; pay CTA; cannot enable/disable |
| Driver | Credit-leave confirm shows `due_at` from terms; reject if disabled/frozen |
| Pegaus admin/support | Permanent disable of relationship or program (ticket + reason + audit) |
| Factory / Payload | No credit policy UI |

## Lifecycle

1. Supplier enables **credit program** (modal ack → `PROGRAM_ENABLE` audit).
2. Supplier enables **retailer relationship** (modal ack → profile `ACTIVE` + terms).
3. Optional: reserve at order create (`CREDIT_RESERVE_AT_CREATE`) — Available = Limit − Balance − Reserved.
4. Driver **credit leave** → same-txn MarkBalance/convert reserve → open AR invoice with DueAt.
5. Aging worker buckets overdue invoices; **dunning step machine** (when `AR_DUNNING_ENABLED`) advances  
   `DUE_SOON → OVERDUE → ESCALATED_1 → ESCALATED_2 → CREDIT_HOLD → COLLECTIONS`, bumps `DelinquencyCount` on first OVERDUE,  
   auto-holds (`FROZEN`) at CREDIT_HOLD without clearing `CreditEnabled`, and fans out inbox + FCM; optionally SMS / email / WhatsApp when `DUNNING_*_PROVIDER` is set (owner keys residual).
6. Repayment clears balance / closes invoice.
7. Permanent disable only via admin API; open AR remains collectible. Re-enable is self-serve again (new ack).

## Money-path gate

```
creditLeave / reserve:
  (if CREDIT_POLICY_V2) program.ProgramEnabled && relationship.CreditEnabled
  && profile.Status == ACTIVE
  && available >= amount
else → reject CREDIT_*; allow cash/card/COD
```

Credit **risk scoring** (`RetailerCreditScores`, score worker, `RiskTier` / suggested limit) is **removed**.
CREDIT_LEAVE and shop-closed timeout use **status + available** only. Limits and freeze/ACTIVE remain.

## Role surfaces

| Surface | Features |
|---------|----------|
| Supplier portal `/credit/policy` | Program + retailer enable (irreversible modal), defaults, hold |
| Supplier portal `/credit/collections` | Limits, freeze, exposure desk |
| Supplier portal `/credit/admin-disable` | Support disable with ticket |
| Warehouse `/credit/policy` | Supplier-scoped read/ops parity |
| Retailer desktop `/credit` | Credit partners + open invoices |
| Driver iOS/Android | Due date on credit-leave success |
| Admin API | `POST /v1/admin/credit-relationships/{supplierId}/{retailerId}/disable` |

## Implemented artifacts

- DDL: `SupplierCreditPrograms`, `RetailerPaymentTerms`, `CreditPolicyAudit`, `OrderCreditReservations`, `ArInvoices`, `ArLedgerEntries`, `ReservedMinor` on profiles.
- Backend: `credit` policy service + reserve/CAS; `ar` invoices + aging; order create no longer blocks cash on missing profile; same-txn MarkBalance on credit leave.
- Clients: `packages/types`, `packages/api-client`, supplier/warehouse/retailer portals, driver due-at copy.
- Flags: `CREDIT_POLICY_V2_ENABLED`, `CREDIT_RESERVE_AT_CREATE`, `AR_INVOICES_ENABLED`, `AR_DUNNING_ENABLED`.
- Ops: `POST /v1/admin/ar/dunning/run-once` (ADMIN) triggers aging + step advance.
- E2E markers: `PX_E2E_CREDIT_ENABLE_IRREVERSIBLE_OK`, `PX_E2E_CREDIT_TERMS_DUE_OK`, `PX_E2E_CREDIT_ADMIN_DISABLE_OK`,  
  `PX_E2E_COLLECTIONS_DUNNING_OK` / `PX_E2E_COLLECTIONS_DUNNING_SKIPPED`.

## Edge case matrix (pass criteria)

| # | Case | Expected |
|---|------|----------|
| 1 | Enable without modal ack | 400 `warning_ack_required` |
| 2 | Self-serve disable after enable | 403 `credit_disable_requires_support` |
| 3 | Admin disable with open AR | Allowed; block new credit; invoices remain |
| 4 | Admin disable with reserved orders | Release/convert per cancel; no new reserve |
| 5 | Global terms change mid-flight | New invoices only; open keep DueDate |
| 6 | Per-retailer override vs global | Override when `UseGlobalDefaults=false` |
| 7 | Retailer not enabled | Credit tender rejected; COD/card OK |
| 8 | Concurrent enable | Idempotent; audit once |
| 9 | Freeze while enabled | Credit leave blocked; visible “On hold” |
| 10 | Over-limit | Reject credit path; cash/card OK |
| 11 | Multi-supplier cart | Credit checked per supplier leg |
| 12 | Credit note on open invoice | Reduces AR; does not disable |
| 13 | Shop-closed credit leave | Same terms/due; frozen → option removed |
| 14 | Negotiation qty after leave | Adjust invoice/AR; terms unchanged |
| 15 | Warehouse without finance perm | 403 on policy APIs |
| 16 | Support disable wrong supplier | IDOR fail closed |
| 17 | Re-enable after admin disable | Self-serve enable + new ack |
| 18 | Program on, zero retailers | No credit leave until retailer enable |
| 19 | Limit lowered below balance | No new credit; existing AR stays |
| 20 | Timezone / calendar days | Supplier TZ for due date (default `Asia/Tashkent`) |

## Explicit non-goals

- Self-serve permanent disable after enable
- Separate warehouse credit ledger
- Consumer BNPL / card installments
- Full ERP GL beyond AR subledger

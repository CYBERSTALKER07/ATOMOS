# Pegasus Payment System Target Architecture

**Date:** 2026-08-19  
**Scope:** `pegasus/`  
**Purpose:** Target business and technical architecture for payment collection, gateway execution, fiscalization, ledger accounting, settlement splitting, and supplier payout.

## 1. Core Principle

Payment collection, gateway execution, fiscalization, ledger accounting, and payout distribution are separate state machines connected by durable events.

```text
retailer order
  → payment session
  → delivery collection
  → gateway capture
  → fiscal receipt
  → immutable ledger
  → supplier / warehouse payout
```

## 2. Order and Payment Authorization

At checkout:

1. Retailer creates an order.
2. Inventory is reserved.
3. The order receives currency and payment policy from `MarketPack`.
4. A multi-supplier cart creates one parent order and one child order per supplier.
5. Each child order keeps its own supplier, warehouse, currency, and payout owner.
6. Checkout never marks the order completed.

Payment states:

```text
CREATED
  → AWAITING_PAYMENT
  → AUTHORIZED
  → CAPTURED
  → FISCALIZING
  → COMPLETED
```

Cash path:

```text
AWAITING_PAYMENT
  → CASH_COLLECTED
  → FISCALIZING
  → COMPLETED
```

## 3. Delivery-Time Collection

Payment is collected after delivery and offload because Pegasus uses pay-at-delivery.

The server calculates:

```text
delivered_gross
− already_captured
− approved_exceptions
= outstanding_due
```

For a cash/card split:

```text
cash_minor >= 0
card_minor >= 0
cash_minor + card_minor == outstanding_due
```

The backend must:

- verify the driver owns the order;
- verify the order is payment-eligible;
- verify currency;
- write cash/card payment legs transactionally;
- use idempotency keys;
- prevent concurrent double collection;
- capture card only after provider confirmation.

A card redirect or authorization is not captured money.

## 4. Gateway Execution

The gateway layer is provider-neutral:

```text
Global Pay / Adyen / Payme / Click / Cash
```

Each provider must implement:

- authorize;
- capture;
- status;
- refund;
- void;
- webhook verification.

Execution flow:

```text
PaymentIntentCreated
  → gateway worker
  → provider request
  → provider webhook or status
  → payment-attempt update
  → ledger settlement
```

Provider failures produce `PAYMENT_FAILED` or `RECONCILIATION_REQUIRED`.

Missing credentials remain an honest `501 no_live_keys`, never a fake checkout URL.

## 5. Immutable Settlement Splits

After the order amount is known, create immutable settlement slices:

```text
gross amount
− platform fee
= net payout
```

Each slice records:

- invoice ID;
- order ID;
- supplier ID;
- warehouse ID;
- payout owner;
- settlement target;
- currency;
- fee policy version;
- fee basis points;
- fee amount;
- net payout amount.

Example:

```text
Order total:      100,000 UZS
Platform fee:       5,000 UZS
Supplier payout:   95,000 UZS
```

For warehouse-local settlement:

```text
Platform fee:       5,000 UZS
Warehouse payout:  95,000 UZS
```

The payout owner comes from the immutable checkout snapshot, not the current policy.

## 6. Treasurer and Ledger

The Treasurer consumes durable payment and order events and writes accounting entries:

```text
debit payment clearing
credit platform fee
credit supplier or warehouse payout
```

Every money movement requires:

- paired ledger entries;
- currency;
- aggregate ID;
- payment/session reference;
- idempotency key;
- immutable audit history.

The Treasurer does not collect payment or call the gateway. It calculates and records financial authority.

## 7. Refunds and Chargebacks

Refunds reverse the original settlement:

```text
refund
  → original supplier/warehouse payout ratio
  → original platform fee ratio
  → gateway refund
  → reversal ledger entries
```

Refunds must not recalculate using the current fee policy.

If the original settlement slice is missing, the operation enters reconciliation review rather than guessing the payout split.

## 8. Fiscal Hard Gate

After payment capture:

```text
CAPTURED
  → FISCALIZING
  → FISCAL_SUCCESS
  → COMPLETED
```

If fiscalization fails:

```text
FISCAL_FAILED
```

The order remains incomplete until fiscal retry succeeds or an authorized operator performs an audited force-complete.

Driver, retailer, and offline clients must never create `COMPLETED` locally.

## 9. Currency Rules

- Currency comes from `MarketPack`.
- Historical orders retain their original currency.
- Payout batches are one supplier, period, and currency.
- Mixed currencies create separate batches.
- FX conversion is not allowed during checkout, payout, refund, or ledger balancing unless explicitly supported by a market adapter.
- Internal amounts are integer minor units only.

## 10. Responsibility Boundaries

| Component | Responsibility |
|---|---|
| Retailer | Requests checkout and confirms payment intent |
| Driver | Confirms delivery and cash/card collection |
| Gateway worker | Calls the PSP and handles provider results |
| Fiscal worker | Creates the legal receipt |
| Treasurer | Splits gross money into ledger accounts |
| Settlement slices | Preserve supplier/warehouse payout ownership |
| Payout service | Generates currency-specific supplier payout batches |
| Reconciliation | Resolves provider and ledger mismatches |
| Platform admin | Controls policies, flags, overrides, and audit |

## 11. Required Safety Invariants

```text
captured_cash
  + captured_card
  + captured_credit
  + approved_exceptions
>= delivered_gross
```

```text
sum(payout_components for one currency)
  == gross_captured - refunds - fees
```

No provider success response alone may complete an order. No event-only payment record may be treated as captured money. No payout batch may contain multiple currencies.

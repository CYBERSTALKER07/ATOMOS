---
name: payme-business-integration
description: 'Implement Payme Business integration in this monorepo. Use for Payme Merchant API handlers, Subscribe API client flows, checkout.paycom.uz links and forms, sandbox and production separation, Uzbek payment flows using Uzcard or HUMO, Paycom JSON-RPC 2.0 request handling, recurring payments, receipts, holds, fiscal data, and reconciliation. Also use when reviewing code so supplier credential management is not confused with customer checkout execution.'
argument-hint: 'Describe the target: merchant API, subscribe API, checkout, recurring payments, mobile, sandbox, production, or supplier config'
---

# Payme Business Integration

## What This Skill Does

Use this skill to implement or review Payme Business integration across backend, web, mobile, and supplier configuration surfaces in this workspace.

This skill keeps four concerns separate:

- Customer payment execution
- Supplier credential configuration
- Merchant API versus Subscribe API
- Sandbox versus production environments

## When to Use

Activate this skill when the request involves any of the following:

- Payme or Paycom checkout flows using `checkout.paycom.uz` or `test.paycom.uz`
- Payme Merchant API methods like `CheckPerformTransaction`, `CreateTransaction`, `PerformTransaction`, `CancelTransaction`, `CheckTransaction`, or `GetStatement`
- Payme Subscribe API methods like `cards.create`, `cards.verify`, `receipts.create`, `receipts.pay`, `receipts.cancel`, or hold flows
- Sandbox versus production setup for Payme Business
- Uzbek payment flows using Uzcard or HUMO
- Mobile Payme integration requirements including the `Powered by Payme` label
- Reviewing whether code is mishandling tiyin amounts, idempotency, fiscal data, or token safety
- Reviewing whether supplier payment settings are incorrectly merged with customer checkout logic

Do not use this skill for unrelated gateways. Do not treat Payme Merchant API and Subscribe API as interchangeable.

## Repo-Specific Guardrails

- Customer checkout belongs in payment execution code, not supplier gateway onboarding.
- Supplier payment credentials remain backend-only and must stay in the vault-backed configuration path.
- This repo already stores shared supplier payment config for `CLICK` and `PAYME`; official Payme Business requirements may require extending that model instead of forcing Payme into the current generic fields.
- Never store raw PAN or card data. Store only Payme-issued tokens when the API allows it.
- `GetStatement` is mandatory for Merchant API implementations and must not be skipped.
- All money amounts must be handled in `tiyin`, not sums.

## Decision Flow

### 1. Classify the Payme surface first

- If Payme calls your backend using JSON-RPC, you are implementing Merchant API.
- If your backend calls Payme using `X-Auth`, you are implementing Subscribe API.
- If the task is hosted checkout, use checkout URL or form initialization.
- If the task is supplier config, treat it as credential storage and operational state, not customer checkout.

### 2. Choose the correct Payme mode

- Merchant API: standard Payme checkout page and Payme-to-merchant callbacks
- Subscribe API: custom forms, tokenized cards, recurring payments, receipts, holds, and invoices

Prefer Subscribe API for custom flows. Use Merchant API for standard checkout.

### 3. Choose the environment explicitly

- `sandbox`: `https://test.paycom.uz` and `https://checkout.test.paycom.uz/api`
- `production`: `https://paycom.uz` and `https://checkout.paycom.uz/api`

Do not mix credentials or URLs across environments.

## Implementation Procedure

### Step 1. Model configuration safely

- Store Payme credentials in backend-only storage.
- Keep sandbox and production credentials as separate records or separately scoped secrets.
- Preserve an explicit distinction between Merchant API credentials and Subscribe API credentials.
- Return only safe summary fields to the frontend.

For this repo, prefer extending backend vault types if official Payme fields do not fit the current shared `merchant_id/service_id/secret_key` structure.

### Step 2. Implement the correct authentication model

- Merchant API: verify `Authorization: Basic base64(login:password)` on inbound Payme requests.
- Merchant API: restrict inbound traffic to documented Payme IP ranges when network topology allows it.
- Subscribe API frontend calls use `X-Auth: {cash_register_id}`.
- Subscribe API backend calls use `X-Auth: {cash_register_id}:{password_key}`.

See [Payme API reference](./references/payme-api.md).

### Step 3. Implement the requested surface

#### Merchant API

- Implement `CheckPerformTransaction`.
- Implement `CreateTransaction`.
- Implement `PerformTransaction`.
- Implement `CancelTransaction`.
- Implement `CheckTransaction`.
- Implement `GetStatement`.
- Implement `SetFiscalData` when fiscalization is in scope.

Merchant API handlers must be idempotent because Payme can retry requests.

#### Subscribe API

- Implement cards lifecycle only when tokenized or recurring payments are actually required.
- Use `receipts.create` and `receipts.pay` for the core payment path.
- Use hold flows only when pre-authorization is a real business requirement.
- Use `receipts.send`, `receipts.check`, `receipts.get`, and `receipts.get_all` for invoice and receipt operations as needed.

#### Hosted checkout

- Build checkout GET or POST initialization using Payme’s documented fields.
- For detailed fiscalized checkout, supply `detail` payload correctly encoded.
- Never trust checkout completion as authoritative until backend transaction state is verified.

#### Mobile

- Use the official mobile SDK pattern where applicable.
- Add the required `Powered by Payme` label.
- Keep sandbox toggles explicit in test builds.

### Step 4. Validate money, errors, and localization

- Validate all amounts in `tiyin`.
- Map Payme error codes to stable backend errors.
- Localize error messages through the `message` object when the Merchant API requires it.
- Preserve enough data for reconciliation and support analysis.

### Step 5. Test in sandbox

- Use sandbox hosts only.
- Run the two mandatory transaction scenarios: unconfirmed cancellation and confirmed cancellation.
- Use documented test cards and fixed SMS code `666666` where required.
- Verify transaction lifecycle, retries, and reconciliation outputs.

See [Sandbox and production guidance](./references/sandbox-production.md).

### Step 6. Cut over to production

- Switch to production hosts only after sandbox flows are stable.
- Use production credentials isolated from sandbox.
- Confirm webhook, statement, and receipt reconciliation before considering the rollout complete.

## Completion Checks

- Merchant API and Subscribe API responsibilities are not mixed.
- Amounts are handled in `tiyin` end-to-end.
- `GetStatement` is implemented for Merchant API.
- Raw card data is never stored.
- Token flows are implemented only when needed.
- Hold flow is not added unless the business process requires pre-authorization.
- Sandbox and production credentials are fully separated.
- Customer checkout logic is not conflated with supplier credential management.
- Idempotency is preserved for retried Payme calls.

## References

- [Payme API reference](./references/payme-api.md)
- [Sandbox and production guidance](./references/sandbox-production.md)
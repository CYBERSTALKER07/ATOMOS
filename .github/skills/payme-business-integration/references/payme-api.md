# Payme API Reference

## Core Concepts

Payme Business supports Uzbek card payments through Uzcard and HUMO.

Main models:

- Merchant API: Payme calls your backend
- Subscribe API: your backend calls Payme

Transport:

- JSON-RPC 2.0 over HTTPS

## Endpoints

### Merchant API

- sandbox host: `https://test.paycom.uz`
- production host: `https://paycom.uz`

### Subscribe API

- sandbox host: `https://checkout.test.paycom.uz/api`
- production host: `https://checkout.paycom.uz/api`

### Hosted checkout

- checkout GET pattern: `https://checkout.paycom.uz/base64(...)`
- sandbox checkout host: `https://test.paycom.uz`

## Authentication

### Merchant API inbound auth

- `Authorization: Basic base64(login:password)`
- Payme requests should be restricted to documented IP ranges when possible:

```text
185.234.113.1 … 185.234.113.15
```

### Subscribe API outbound auth

- frontend: `X-Auth: {cash_register_id}`
- backend: `X-Auth: {cash_register_id}:{password_key}`

## Merchant API Methods

Required core methods:

- `CheckPerformTransaction`
- `CreateTransaction`
- `PerformTransaction`
- `CancelTransaction`
- `CheckTransaction`
- `GetStatement`

Optional or recommended:

- `SetFiscalData`

### CheckPerformTransaction

Use to decide whether payment is allowed.

Important request detail:

- `amount` is in `tiyin`
- account fields identify your internal object, for example `order_id`

### GetStatement

Mandatory for reconciliation.

Do not ship Merchant API without it.

## Subscribe API Methods

### Cards

- `cards.create`
- `cards.get_verify_code`
- `cards.verify`
- `cards.check`
- `cards.remove`

### Receipts

- `receipts.create`
- `receipts.pay`
- `receipts.send`
- `receipts.cancel`
- `receipts.check`
- `receipts.get`
- `receipts.get_all`
- `receipts.set_fiscal_data`

### Hold Flow

Use `hold: true` in `receipts.create` and `receipts.pay`, then:

- `receipts.confirm_hold`
- `receipts.cancel`

### Important Receipt States

- `0`: created
- `4`: paid
- `5`: held
- `21`: queued for cancellation
- `50`: cancelled

## Checkout Initialization

### GET format

```text
https://checkout.paycom.uz/base64(m=ID;ac.order_id=197;a=500)
```

### POST form

Typical fields:

- `merchant`
- `amount`
- `account[order_id]`
- optional `lang`
- optional `callback`
- optional `callback_timeout`
- optional `description`
- optional `detail`

For fiscalization, `detail` can include items, shipping, VAT, IKPU codes, units, and package codes.

## Mobile Integration

Android SDK package is documented as:

```gradle
compile 'uz.paycom:payment:$latest'
```

Operational rules:

- show `Powered by Payme`
- use sandbox toggle in test builds
- never store raw PAN on your backend

## Common Error Themes

- `-31001`: invalid amount
- `-31003`: transaction not found
- `-31007`: cannot cancel because service already provided
- `-31008`: operation impossible in current state
- `-31050...99`: invalid account fields
- `-31601`: merchant not found or blocked
- `-31630`: card error

Map these into stable internal errors without losing the original Payme code.

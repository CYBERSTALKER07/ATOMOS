# CLICK API Reference

## Core Endpoints

- Hosted checkout: `https://my.click.uz/services/pay`
- Embedded checkout script: `https://my.click.uz/pay/checkout.js`
- Merchant API base: `https://api.click.uz/v2/merchant/`
- Merchant portal: `http://merchant.click.uz`

## Merchant API Authentication

Header format:

```text
Auth: merchant_user_id:digest:timestamp
```

Rules:

- `timestamp`: current UNIX timestamp in seconds
- `digest`: `sha1(timestamp + secret_key)`
- required headers: `Accept`, `Content-Type`, `Auth`

## Hosted Web Redirect

Required query params:

- `service_id`
- `merchant_id`
- `amount`
- `transaction_param`

Optional query params:

- `merchant_user_id`
- `return_url`
- `card_type`

Example:

```text
https://my.click.uz/services/pay?service_id=123&merchant_id=456&amount=15000.00&transaction_param=ORD-789
```

## Embedded Checkout

Use `createPaymentRequest` from `checkout.js`.

Callback status meanings:

- `status < 0`: error
- `status = 0`: payment created
- `status = 1`: processing
- `status = 2`: success

These statuses are not a substitute for backend reconciliation.

## Merchant API Operations

### Create invoice

- `POST /invoice/create`
- payload fields: `service_id`, `amount`, `phone_number`, `merchant_trans_id`

### Invoice status

- `GET /invoice/status/:service_id/:invoice_id`

### Payment status by payment id

- `GET /payment/status/:service_id/:payment_id`

### Payment status by merchant transaction id

- `GET /payment/status_by_mti/:service_id/:merchant_trans_id/YYYY-MM-DD`

### Reversal

- `DELETE /payment/reversal/:service_id/:payment_id`

Constraints:

- payment must already be successful
- prior-month reversals are restricted
- reversal can still be declined upstream

### Card tokenization

1. `POST /card_token/request`
2. `POST /card_token/verify`
3. `POST /card_token/payment`
4. `DELETE /card_token/:service_id/:card_token`

## Mobile Deeplinks

Android and iOS both launch the standard hosted checkout URL.

Recommended behavior:

- include `transaction_param`
- optionally include `return_url`
- on app resume or deep link return, query your backend using the original transaction id

## Emulator and Registration

Before production:

1. configure reachable `Prepare URL` and `Complete URL`
2. enter `service_id`, `merchant_user_id`, `secret_key`, and a representative `merchant_trans_id`
3. pass all mandatory scenarios
4. generate the report
5. verify registration state at `merchant.click.uz`
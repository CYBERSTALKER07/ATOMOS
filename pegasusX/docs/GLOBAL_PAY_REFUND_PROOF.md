# Global Pay refund (`RF`) proof

**Date:** 2026-08-12  
**Gap:** P2-10  
**Status:** Wire format + executor path proven against the in-repo Global Pay simulator. Live merchant gateway confirm remains an ops checklist when merchant docs/creds are available.

## Proven in CI

`payment.TestGlobalPayRefundAgainstSimulator`:

1. Auth against simulator `/v1/merchant/auth`
2. Backoffice `POST /payments/v2/payment/{id}/perform` with **`action: "RF"`** (default; override `GLOBAL_PAY_REFUND_ACTION`)
3. Amount/currency forwarded; provider ref returned

Simulator accepts `CP` (capture) and `RF` (refund) only.

Product path: `Service.RefundCardPayment` / chargeback partial refund → `ExecutionActionRefund` → `globalpayProviderExecutor.executeRefund`.

## Live merchant confirm (ops)

When Global Pay merchant support confirms the perform action code:

```bash
# default is already RF; set only if merchant docs differ
export GLOBAL_PAY_REFUND_ACTION=RF
export GLOBAL_PAY_ENV=sandbox   # not stub
# real username/password/service id
go test ./payment/ -run TestGlobalPayRefundAgainstSimulator  # still sim
# live: exercise RefundCardPayment against sandbox order in a controlled soak
```

Until then, **do not treat stub-mode refunds** (`gp_refund_stub_*`) as gateway proof — stub only runs when credentials are empty and stub mode is on.

## Non-goals

- Claiming production refund success without a sandbox/live perform response
- Changing ledger semantics (ledger debit still happens; gateway soft-fail surfaces for ops)

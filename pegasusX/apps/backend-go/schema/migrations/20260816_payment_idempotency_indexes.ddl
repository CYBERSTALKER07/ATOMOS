-- Phase 0 money-path hardening: financial idempotency is a database guarantee,
-- not only a Redis TTL layer.
--
-- 1) One payment leg per logical operation: OrderPaymentLegs.IdempotencyKey is
--    stable per capture/collect/credit operation (card-capture-<order>,
--    card-backorder-<order>, cash-<order>-*, credit-leave-<order>). A duplicate
--    insert now aborts the transaction instead of double-recording money.
CREATE UNIQUE INDEX Idx_OrderPaymentLegs_IdempotencyKey
  ON OrderPaymentLegs(IdempotencyKey);

-- 2) One ledger entry per provider-side fact: the same gateway + entry type +
--    provider reference can never be recorded twice (webhook redelivery with the
--    same transaction id, replayed chargeback, etc.). NULL-filtered because
--    internal entries without a provider reference are exempt.
CREATE UNIQUE NULL_FILTERED INDEX Idx_PaymentLedgerEntries_GatewayTypeRef
  ON PaymentLedgerEntries(Gateway, EntryType, ReferenceId);

-- Tracks per-order inventory reservation for idempotent backfill of legacy scheduled pre-orders.
CREATE TABLE OrderStockReservationMarkers (
  OrderId     STRING(36) NOT NULL,
  ReservedAt  TIMESTAMP NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (OrderId);

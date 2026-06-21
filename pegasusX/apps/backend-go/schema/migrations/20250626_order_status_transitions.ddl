-- Immutable order status audit trail (delays, promotions, lifecycle).
CREATE TABLE OrderStatusTransitions (
  OrderId          STRING(36)  NOT NULL,
  TransitionId     STRING(36)  NOT NULL,
  PreviousStatus   STRING(32),
  NewStatus        STRING(32)  NOT NULL,
  Reason           STRING(512),
  ActorRole        STRING(32),
  ActorId          STRING(128),
  EventKind        STRING(32),
  MetadataJson     BYTES(MAX),
  CreatedAt        TIMESTAMP   NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (OrderId, CreatedAt DESC, TransitionId);

CREATE INDEX Idx_OrderStatusTransitions_ByOrder ON OrderStatusTransitions(OrderId, CreatedAt DESC);

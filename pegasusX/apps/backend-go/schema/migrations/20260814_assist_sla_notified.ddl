-- Wave C4.1: Assist SLA breach idempotency marker.
-- Worker notifies once when OPEN ticket is past SlaDueAt.
-- Flag: ASSIST_SLA_ENABLED (default off). SLA minutes: ASSIST_SLA_MINUTES (default 15) or pack config.

ALTER TABLE RetailerAssistanceTickets ADD COLUMN SlaBreachNotifiedAt TIMESTAMP;

CREATE INDEX Idx_RetailerAssist_BySlaDue
  ON RetailerAssistanceTickets (Status, SlaDueAt);

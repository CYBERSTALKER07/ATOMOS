-- Wave C4.1: assist SLA breach worker
ALTER TABLE RetailerAssistanceTickets ADD COLUMN SlaBreachedAt TIMESTAMP;

CREATE NULL_FILTERED INDEX Idx_RetailerAssist_SlaDue
  ON RetailerAssistanceTickets(Status, SlaDueAt)
  WHERE Status = 'OPEN';

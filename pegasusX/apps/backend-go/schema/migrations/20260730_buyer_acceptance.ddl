ALTER TABLE Orders ADD COLUMN BuyerAcceptanceStatus STRING(MAX);
ALTER TABLE Orders ADD COLUMN BuyerAcceptanceDeadline TIMESTAMP;

CREATE TABLE ExceptionTickets (
    TicketId STRING(36) NOT NULL,
    Type STRING(64) NOT NULL,
    OrderId STRING(36) NOT NULL,
    EhfId STRING(64),
    Severity STRING(16) NOT NULL,
    Status STRING(32) NOT NULL,
    Title STRING(256) NOT NULL,
    Description STRING(MAX),
    AssignedRole STRING(64),
    CreatedAt TIMESTAMP NOT NULL OPTIONS (allow_commit_timestamp=true),
    CreatedBy STRING(128),
    Payload JSON
) PRIMARY KEY (TicketId);

CREATE INDEX Idx_Orders_BuyerAcceptance 
ON Orders(FiscalStatus, BuyerAcceptanceStatus, BuyerAcceptanceDeadline);

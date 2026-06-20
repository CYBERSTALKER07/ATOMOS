-- Warehouse delivery-date proposal negotiation (retailer accept/reject).

ALTER TABLE Orders ADD COLUMN ProposedDeliveryDate TIMESTAMP;
ALTER TABLE Orders ADD COLUMN DeliveryProposalAt TIMESTAMP;
ALTER TABLE Orders ADD COLUMN DeliveryProposalBy STRING(128);
ALTER TABLE Orders ADD COLUMN DeliveryProposalReason STRING(512);

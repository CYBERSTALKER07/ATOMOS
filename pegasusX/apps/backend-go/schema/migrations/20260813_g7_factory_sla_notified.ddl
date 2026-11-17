-- G7.1: idempotent FACTORY_SLA_BREACH notify marker on supply requests.
ALTER TABLE WarehouseSupplyRequests ADD COLUMN SlaBreachNotifiedAt TIMESTAMP;

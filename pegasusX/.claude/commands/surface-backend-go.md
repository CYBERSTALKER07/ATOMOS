# Surface Skill: Backend Go & Distributed Services

Governs Go 1.25+, Cloud Spanner transactional consistency, Outbox event streaming to Kafka, Chi router, and zero race conditions under `go test -race`.
Enforce:
- Cloud Spanner transactions (`spanner.ReadWriteTransaction` retry idempotency).
- OutboxEvents table writes inside the transaction; zero direct HTTP-to-Kafka publishing.
- `defer iter.Stop()` on all RowIterators.
- 100% tenant isolation with `SupplierId` partitioning.

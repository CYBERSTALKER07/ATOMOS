// Package outbox implements the V.O.I.D. Transactional Outbox pattern.
//
// The outbox guarantees atomicity between Spanner entity mutations and the
// Kafka events they are supposed to emit. Without it, the naive sequence
// "commit row → publish event" has a race: if the process dies between the
// two steps, the database says the entity exists but no consumer ever hears
// about it — a "ghost" entity. The outbox removes that race by making the
// event durable in the same transaction as the entity:
//
//  1. Handler opens a spanner.ReadWriteTransaction.
//  2. Handler writes the domain row AND calls outbox.Emit(txn, Event{...}).
//  3. Commit either persists both or neither.
//  4. The Relay goroutine (started from bootstrap.NewApp) tails unpublished
//     OutboxEvents rows, publishes them to the declared topic via the Kafka
//     writer with RequiredAcks=all, and marks PublishedAt on success.
//
// Event.AggregateID is used as the Kafka message key so per-entity ordering
// is preserved across partitions. Event.TopicName is the destination topic.
// Event.Payload is already JSON-encoded bytes — the outbox does not shape
// the payload beyond passing it through.
//
// Direct writer.WriteMessages calls from handlers are reserved for tolerant,
// non-durable traffic (telemetry, location pings). Anything that represents
// a state transition MUST flow through the outbox.
package outbox

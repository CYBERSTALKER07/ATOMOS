// Package telemetryaudit owns best-effort Kafka journaling and Spanner audit
// persistence for live driver GPS pings. It complements the in-memory telemetry
// hub without changing the JSON WebSocket contract current clients use.
package telemetryaudit

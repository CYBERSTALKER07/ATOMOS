// Package notifications owns the persistent notification inbox backed by the
// Notifications Spanner table. It provides CRUD for notification records,
// pure formatting functions for event-to-notification mapping, and pluggable
// transport delivery (WebSocket, FCM/APNs stubs). It does NOT own the Kafka
// consumer that triggers notifications (see kafka/notification_dispatcher).
package notifications

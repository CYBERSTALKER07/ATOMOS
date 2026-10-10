/**
 * Module: Messaging
 * Output declarations
 */

output "kafka_cluster_id" {
  description = "The ID of the Managed Kafka cluster."
  value       = google_managed_kafka_cluster.events.cluster_id
}

output "kafka_cluster_name" {
  description = "The full resource name of the Managed Kafka cluster."
  value       = google_managed_kafka_cluster.events.id
}

output "kafka_cluster_create_time" {
  description = "The creation timestamp of the Managed Kafka cluster."
  value       = google_managed_kafka_cluster.events.create_time
}

output "kafka_topics" {
  description = "List of created canonical Kafka topic IDs."
  value       = [for t in google_managed_kafka_topic.canonical_topics : t.topic_id]
}

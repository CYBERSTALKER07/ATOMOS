variable "project_id" {
  description = "Google Cloud project ID for pegasusX workloads."
  type        = string
}

variable "tenant_slug" {
  description = "Client or sandbox slug used to namespace isolated SSMR resources."
  type        = string
  default     = "ssmr"
}

variable "resource_prefix" {
  description = "Explicit resource prefix override. When empty, terraform uses pegasusx-<tenant_slug>."
  type        = string
  default     = ""
}

variable "region" {
  description = "Primary deployment region."
  type        = string
  default     = "asia-south1"
}

variable "environment" {
  description = "Environment label for resource tagging."
  type        = string
  default     = "dev"
}

variable "vpc_name" {
  description = "VPC name used by backend workloads and Memorystore."
  type        = string
  default     = ""
}

variable "spanner_instance_name" {
  description = "Cloud Spanner instance name."
  type        = string
  default     = ""
}

variable "spanner_database_name" {
  description = "Cloud Spanner database name."
  type        = string
  default     = ""
}

variable "spanner_display_name" {
  description = "Cloud Spanner display name. When empty, derived from the tenant slug."
  type        = string
  default     = ""
}

variable "redis_instance_name" {
  description = "Memorystore Redis instance name."
  type        = string
  default     = ""
}

variable "redis_memory_size_gb" {
  description = "Memorystore Redis memory size in GB."
  type        = number
  default     = 1
}

variable "kafka_bootstrap_servers" {
  description = "Kafka bootstrap servers for app env (stored in Secret Manager)."
  type        = string
  default     = ""
  sensitive   = true
}

variable "kafka_topic_main" {
  description = "Default Kafka topic used by backend-go outbox relay for order and state events."
  type        = string
  default     = "ssmr.events.orders"
}

variable "kafka_topic_spatial" {
  description = "Kafka topic reserved for spatial or H3 fanout workloads."
  type        = string
  default     = "ssmr.events.spatial"
}

variable "kafka_topic_realtime" {
  description = "Kafka topic reserved for realtime socket and fleet fanout."
  type        = string
  default     = "ssmr.events.realtime"
}

variable "kafka_topic_webhooks" {
  description = "Kafka topic reserved for outbound webhook delivery work."
  type        = string
  default     = "ssmr.events.webhooks"
}

variable "firebase_project_id" {
  description = "Firebase project id for ID token verification."
  type        = string
  default     = ""
}

variable "firebase_auth_enabled" {
  description = "Whether Firebase bearer token verification is enabled at runtime."
  type        = bool
  default     = false
}

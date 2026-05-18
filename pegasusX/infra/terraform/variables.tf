variable "project_id" {
  description = "Google Cloud project ID for pegasusX workloads."
  type        = string
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
  default     = "pegasusx-vpc"
}

variable "spanner_instance_name" {
  description = "Cloud Spanner instance name."
  type        = string
  default     = "pegasusx-ledger-instance"
}

variable "spanner_database_name" {
  description = "Cloud Spanner database name."
  type        = string
  default     = "pegasusx-db"
}

variable "redis_instance_name" {
  description = "Memorystore Redis instance name."
  type        = string
  default     = "pegasusx-redis"
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
  description = "Default Kafka topic used by backend-go outbox relay."
  type        = string
  default     = "pegasusx-main"
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

/**
 * Module: Messaging
 * Variable declarations for Google Managed Service for Apache Kafka cluster and canonical topics.
 */

variable "project_id" {
  description = "The Google Cloud Platform project ID."
  type        = string
}

variable "region" {
  description = "The GCP region for the Managed Kafka cluster."
  type        = string
  default     = "europe-west3"
}

variable "kafka_cluster_id" {
  description = "Unique identifier for the Managed Apache Kafka cluster."
  type        = string
  default     = "pegasusx-events-cluster"
}

variable "subnet_id" {
  description = "The subnetwork ID or self link where the Kafka broker endpoints will be attached."
  type        = string
}

variable "kafka_vcpu_count" {
  description = "The number of vCPUs allocated per broker in the Managed Kafka cluster."
  type        = number
  default     = 3
}

variable "kafka_memory_bytes" {
  description = "Memory allocated per broker in bytes (16 GiB = 17179869184 bytes)."
  type        = number
  default     = 17179869184
}

variable "topics" {
  description = "Map of canonical partitioned Kafka topics to create with partition and replication configuration."
  type = map(object({
    partition_count    = number
    replication_factor = number
    configs            = optional(map(string), {})
  }))
  default = {
    "pegasusx-main" = {
      partition_count    = 12
      replication_factor = 3
      configs = {
        "cleanup.policy" = "delete"
        "retention.ms"   = "604800000" # 7 days
      }
    }
    "pegasusx-orders" = {
      partition_count    = 12
      replication_factor = 3
      configs = {
        "cleanup.policy" = "delete"
        "retention.ms"   = "604800000" # 7 days
      }
    }
    "pegasusx-dispatch" = {
      partition_count    = 6
      replication_factor = 3
      configs = {
        "cleanup.policy" = "delete"
        "retention.ms"   = "604800000" # 7 days
      }
    }
    "pegasusx-realtime" = {
      partition_count    = 12
      replication_factor = 3
      configs = {
        "cleanup.policy" = "delete"
        "retention.ms"   = "86400000" # 1 day
      }
    }
    "pegasusx-demand" = {
      partition_count    = 6
      replication_factor = 3
      configs = {
        "cleanup.policy" = "delete"
        "retention.ms"   = "604800000" # 7 days
      }
    }
    "logistics.exceptions.v1" = {
      partition_count    = 6
      replication_factor = 3
      configs = {
        "cleanup.policy" = "delete"
        "retention.ms"   = "2592000000" # 30 days
      }
    }
    "logistics.telemetry.v1" = {
      partition_count    = 6
      replication_factor = 3
      configs = {
        "cleanup.policy" = "delete"
        "retention.ms"   = "604800000" # 7 days
      }
    }
    "pegasusx-freeze-locks" = {
      partition_count    = 6
      replication_factor = 3
      configs = {
        "cleanup.policy" = "compact"
      }
    }
    "pegasusx-inventory-import" = {
      partition_count    = 6
      replication_factor = 3
      configs = {
        "cleanup.policy" = "delete"
        "retention.ms"   = "259200000" # 3 days
      }
    }
    "pegasusx-main-dlq" = {
      partition_count    = 6
      replication_factor = 3
      configs = {
        "cleanup.policy" = "delete"
        "retention.ms"   = "1209600000" # 14 days
      }
    }
  }
}

variable "environment" {
  description = "The deployment environment (e.g. production, staging, ssmr)."
  type        = string
  default     = "production"
}

variable "labels" {
  description = "Resource labels applied to messaging infrastructure."
  type        = map(string)
  default = {
    managed_by = "terraform"
    system     = "pegasusx"
    tier       = "messaging"
  }
}

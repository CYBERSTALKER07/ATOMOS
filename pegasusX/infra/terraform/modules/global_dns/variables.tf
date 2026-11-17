variable "project_id" {
  description = "Google Cloud project ID hosting the global Cloud DNS managed zone."
  type        = string
}

variable "zone_name" {
  description = "Resource name of the public Cloud DNS managed zone."
  type        = string
  default     = "pegasusx-app"
}

variable "dns_name" {
  description = "Trailing-dot DNS name for the managed zone."
  type        = string
  default     = "pegasusx.app."
}

variable "records" {
  description = "Per-cell public API records. Placeholder IPs until a cell LB exists."
  type = map(object({
    type    = string
    ttl     = number
    rrdatas = list(string)
  }))
}

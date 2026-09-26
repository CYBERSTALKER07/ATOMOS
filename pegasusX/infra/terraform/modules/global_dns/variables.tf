variable "project_id" {
  type = string
}

variable "zone_name" {
  type    = string
  default = "pegasusx-app"
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

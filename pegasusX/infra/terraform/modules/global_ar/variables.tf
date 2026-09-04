variable "project_id" {
  type = string
}

variable "location" {
  description = "Multi-region or region for the shared image repo (not a cell stack)."
  type        = string
  default     = "europe"
}

variable "repository_id" {
  type    = string
  default = "pegasusx"
}

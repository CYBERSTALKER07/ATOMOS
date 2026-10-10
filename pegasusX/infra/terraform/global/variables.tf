variable "project_id" {
  description = "Global control-plane project. Must not be pegasus-503013 or pegasusx-cell-eu."
  type        = string
}

variable "region" {
  description = "Provider default region (AR uses multi-region europe)."
  type        = string
  default     = "europe-west1"
}

variable "terraform_state_prefix" {
  description = "Must match backend.hcl (pegasusx/global)."
  type        = string
}

variable "uz_api_ipv4" {
  description = "RFC 5737 documentation IP until the UZ ingress LB exists. Replace before apply."
  type        = string
  default     = "203.0.113.10"
}

variable "eu_api_ipv4" {
  description = "RFC 5737 documentation IP until the EU ingress LB exists. Replace before apply."
  type        = string
  default     = "203.0.113.20"
}

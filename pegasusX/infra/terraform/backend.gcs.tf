# GCS Remote State configuration for pegasusX Terraform.
# The bucket name will be updated when provisioning the new GCP infrastructure.
# Prefix is configured via -backend-config=backend-ssmr.hcl (e.g. pegasusx/ssmr).

terraform {
  backend "gcs" {
    bucket = "pegasusx-prod-uz-tfstate"
  }
}

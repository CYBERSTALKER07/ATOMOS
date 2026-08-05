# Remote Terraform state (Gate-0). Bucket created 2026-08-05 on pegasus-503013.
terraform {
  backend "gcs" {
    bucket = "pegasus-503013-terraform-state"
    prefix = "pegasusx/ssmr"
  }
}

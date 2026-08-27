# Remote Terraform state (Gate-0). Bucket created 2026-08-05 on pegasus-503013.
# Prefix is per-cell and MUST come from -backend-config (backend-ssmr.hcl or a
# cell-specific hcl). Hardcoding prefix = "pegasusx/ssmr" here would let an
# europe-west1 init mutate the live UZ/SSMR state (GS-C1 forbid).
terraform {
  backend "gcs" {
    bucket = "pegasus-503013-terraform-state"
  }
}

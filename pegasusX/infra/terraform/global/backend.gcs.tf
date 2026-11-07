# Global-plane state. Prefix MUST come from backend.hcl (pegasusx/global).
# MUST NOT be pegasusx/ssmr (live UZ) or pegasusx/cell-eu (EU stack).
terraform {
  backend "gcs" {
    bucket = "pegasus-503013-terraform-state"
  }
}

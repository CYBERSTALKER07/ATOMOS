# Project-factory state. Prefix comes from backend.hcl (pegasusx/cell-eu-project).
# MUST NOT be pegasusx/ssmr (live cell) or pegasusx/cell-eu (cell stack).
terraform {
  backend "gcs" {
    bucket = "pegasus-503013-terraform-state"
  }
}

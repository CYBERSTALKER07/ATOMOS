# Optional remote state — copy to backend.tf and set bucket name.
# Per-cell prefix belongs in a backend-*.hcl passed to terraform init
# -backend-config (see backend-ssmr.hcl / backend-cell.example.hcl).
# Do not hardcode prefix = "pegasusx/ssmr" in the terraform block.
#
#   gsutil mb -l asia-south1 gs://your-terraform-state-bucket
#   gsutil versioning set on gs://your-terraform-state-bucket

terraform {
  backend "gcs" {
    bucket = "your-terraform-state-bucket"
  }
}

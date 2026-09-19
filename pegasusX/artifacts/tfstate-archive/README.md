# Terraform state archives (historical void-494000 snapshots) are **not** kept in git.
#
# Remote state: `infra/terraform/backend.gcs.tf` →
#   gs://pegasus-503013-terraform-state/pegasusx/ssmr
#
# If you need a local forensic copy, store it outside the repo (encrypted) and
# rotate any secrets that appeared in old `staging.tfvars` / `.env.k8s.generated`
# snapshots (jwt_secret, maps key, GP webhook secret).

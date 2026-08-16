#!/usr/bin/env bash
# GS-C1: applying europe-west1 must not be able to open pegasusx/ssmr state.
# File-level proof only — does not terraform init/plan/apply against live GCS.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
TF="$ROOT/infra/terraform"
fail=0

die() {
  echo "FAIL: $*" >&2
  fail=1
}

if grep -E '^[[:space:]]*prefix[[:space:]]*=' "$TF/backend.gcs.tf" >/dev/null; then
  die "backend.gcs.tf must not set a state prefix (use backend-*.hcl)"
fi

if ! grep -E '^[[:space:]]*prefix[[:space:]]*=[[:space:]]*"pegasusx/ssmr"' "$TF/backend-ssmr.hcl" >/dev/null; then
  die "backend-ssmr.hcl must keep prefix = \"pegasusx/ssmr\" for the live cell"
fi

if grep -E '^[[:space:]]*prefix[[:space:]]*=[[:space:]]*"pegasusx/ssmr"' "$TF/backend-cell.example.hcl" >/dev/null; then
  die "backend-cell.example.hcl must not use prefix pegasusx/ssmr"
fi

if ! grep -E '^[[:space:]]*prefix[[:space:]]*=[[:space:]]*"pegasusx/cell-eu"' "$TF/backend-cell.example.hcl" >/dev/null; then
  die "backend-cell.example.hcl must declare a different prefix (pegasusx/cell-eu)"
fi

if ! grep -E 'variable "cell_id"' "$TF/cell.tf" >/dev/null; then
  die "cell.tf must declare variable cell_id"
fi

if ! grep -E 'variable "api_hostname"' "$TF/cell.tf" >/dev/null; then
  die "cell.tf must declare variable api_hostname"
fi

if ! grep -E 'variable "k8s_namespace"' "$TF/cell.tf" >/dev/null; then
  die "cell.tf must declare variable k8s_namespace"
fi

if ! grep -E 'europe_west1_cannot_use_ssmr_state' "$TF/cell.tf" >/dev/null; then
  die "cell.tf must assert europe-west1 cannot use pegasusx/ssmr"
fi

if grep -F 'svc.id.goog[pegasusx/backend-go]' "$TF/gke.tf" >/dev/null; then
  die "gke.tf Workload Identity must read k8s_namespace, not hardcode pegasusx/backend-go"
fi

if ! grep -F 'local.k8s_namespace' "$TF/gke.tf" >/dev/null; then
  die "gke.tf Workload Identity must use local.k8s_namespace"
fi

if [[ ! -f "$ROOT/infra/k8s/overlays/cells/uz/kustomization.yaml" ]]; then
  die "missing infra/k8s/overlays/cells/uz/kustomization.yaml"
fi

if ! grep -F -- '-backend-config=backend-ssmr.hcl' "$ROOT/Makefile" >/dev/null; then
  die "Makefile terraform-init must pass -backend-config=backend-ssmr.hcl"
fi

if ! grep -F 'Do not use Terraform workspaces' "$TF/README.md" >/dev/null; then
  die "terraform README must forbid workspaces as the cell strategy"
fi

if [[ ! -f "$TF/cells/uz/backend.hcl" || ! -f "$TF/cells/uz/cell.tfvars" ]]; then
  die "missing infra/terraform/cells/uz/{backend.hcl,cell.tfvars}"
fi

if [[ ! -f "$TF/cells/eu/backend.hcl" || ! -f "$TF/cells/eu/cell.tfvars" ]]; then
  die "missing infra/terraform/cells/eu/{backend.hcl,cell.tfvars}"
fi

if ! grep -E '^[[:space:]]*prefix[[:space:]]*=[[:space:]]*"pegasusx/ssmr"' "$TF/cells/uz/backend.hcl" >/dev/null; then
  die "cells/uz/backend.hcl must keep prefix pegasusx/ssmr"
fi

if grep -E '^[[:space:]]*prefix[[:space:]]*=[[:space:]]*"pegasusx/ssmr"' "$TF/cells/eu/backend.hcl" >/dev/null; then
  die "cells/eu/backend.hcl must not use prefix pegasusx/ssmr"
fi

if ! grep -E '^[[:space:]]*prefix[[:space:]]*=[[:space:]]*"pegasusx/cell-eu"' "$TF/cells/eu/backend.hcl" >/dev/null; then
  die "cells/eu/backend.hcl must set prefix pegasusx/cell-eu"
fi

if grep -E '^[[:space:]]*project_id[[:space:]]*=[[:space:]]*"pegasus-503013"' "$TF/cells/eu/cell.tfvars" >/dev/null; then
  die "cells/eu/cell.tfvars must not use project pegasus-503013"
fi

if ! grep -E 'non_uz_cell_not_in_live_project' "$TF/cell.tf" >/dev/null; then
  die "cell.tf must reject a non-uz cell in pegasus-503013"
fi

if ! grep -F 'cell-plan' "$ROOT/Makefile" >/dev/null; then
  die "Makefile must define cell-plan"
fi

if [[ ! -f "$ROOT/infra/k8s/overlays/cells/eu/kustomization.yaml" ]]; then
  die "missing infra/k8s/overlays/cells/eu/kustomization.yaml"
fi

if ! grep -E 'FISCAL_ALLOW_COMMERCIAL_RECEIPTS=true' "$ROOT/infra/k8s/overlays/cells/eu/kustomization.yaml" >/dev/null; then
  die "cells/eu overlay must allow commercial fiscal"
fi
if grep -E 'FISCAL_PROVIDER=MY_SOLIQ' "$ROOT/infra/k8s/overlays/cells/eu/kustomization.yaml" >/dev/null; then
  die "cells/eu overlay must not set MY_SOLIQ"
fi
if ! grep -E 'SPANNER_PROJECT=pegasusx-cell-eu' "$ROOT/infra/k8s/overlays/cells/eu/kustomization.yaml" >/dev/null; then
  die "cells/eu overlay must point Spanner at pegasusx-cell-eu"
fi

if [[ ! -f "$TF/cells/eu/project/backend.hcl" || ! -f "$TF/cells/eu/project/main.tf" ]]; then
  die "missing cells/eu/project factory (GS-C3)"
fi
if ! grep -E '^[[:space:]]*prefix[[:space:]]*=[[:space:]]*"pegasusx/cell-eu-project"' "$TF/cells/eu/project/backend.hcl" >/dev/null; then
  die "project-factory backend must be pegasusx/cell-eu-project"
fi
if grep -E '^[[:space:]]*prefix[[:space:]]*=[[:space:]]*"pegasusx/ssmr"' "$TF/cells/eu/project/backend.hcl" >/dev/null; then
  die "project-factory must not use pegasusx/ssmr"
fi
if ! grep -E 'non_uz_forbids_uz_restore' "$TF/cell.tf" >/dev/null; then
  die "cell.tf must forbid UZ backup restore on a non-uz cell"
fi
if ! grep -F 'cell-project-plan' "$ROOT/Makefile" >/dev/null; then
  die "Makefile must define cell-project-plan"
fi
if ! grep -E 'forbids Spanner backup restore' "$ROOT/scripts/cell_migrate.sh" >/dev/null; then
  die "cell_migrate.sh must refuse UZ restore"
fi

if [[ ! -f "$TF/global/backend.hcl" || ! -f "$TF/global/main.tf" ]]; then
  die "missing infra/terraform/global (GS-C5)"
fi
if ! grep -E '^[[:space:]]*prefix[[:space:]]*=[[:space:]]*"pegasusx/global"' "$TF/global/backend.hcl" >/dev/null; then
  die "global backend must be pegasusx/global"
fi
if grep -E '^[[:space:]]*prefix[[:space:]]*=[[:space:]]*"pegasusx/ssmr"' "$TF/global/backend.hcl" >/dev/null; then
  die "global backend must not use pegasusx/ssmr"
fi
if grep -E '^[[:space:]]*project_id[[:space:]]*=[[:space:]]*"pegasus-503013"' "$TF/global/global.tfvars" >/dev/null; then
  die "global plane must not use pegasus-503013"
fi
if [[ ! -f "$TF/modules/global_dns/main.tf" || ! -f "$TF/modules/global_ar/main.tf" ]]; then
  die "missing modules/global_dns or modules/global_ar"
fi
if ! grep -F 'global-plan' "$ROOT/Makefile" >/dev/null; then
  die "Makefile must define global-plan"
fi

if [[ "$fail" -ne 0 ]]; then
  echo "GS-C cell isolation guard failed" >&2
  exit 1
fi

echo "GS-C cell isolation: live prefix pegasusx/ssmr is uz/ssmr only; cells/eu uses pegasusx/cell-eu and project pegasusx-cell-eu"

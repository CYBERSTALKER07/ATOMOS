|.PHONY: help infra-up infra-down ssmr-infra-up ssmr-infra-down test-ssmr-infra test-ssmr-lifecycle test-ssmr-fiscal ssmr-ecosystem-marker-gate money-path-gate phase1-gate planning-training-export planning-export-validate planning-export-local-cron planning-export-audit-gate planning-export-audit-bootstrap fire-drill-ssmr phase0-preflight phase0-plan phase0-apply phase0-sync-secrets phase0-migrate load-cert load-cert-ssmr load-cert-cloud cloud-smoke-ssmr parity-contract parity-contract-full gap-hunter-gate gen-contracts-gate partner-openapi-gate jwt-openapi-gate schema-drift-gate apply-shop-closed-ddl px12-preflight p0-preflight p1-pilot-weekly wire-ready qa-gate validate-ai-worker-k8s validate-backend-k8s validate-desktop-clients validate-launch-readiness validate-production-profile docker-build docker-build-backend docker-build-ai-worker backend-run backend-build supplier-portal-dev marketing-site-dev

help:
	@echo "pegasusX make targets"
	@echo "  infra-up              start Spanner emulator, Redis, Kafka via docker-compose"
	@echo "  infra-down            stop infra"
	@echo "  ssmr-infra-up         start the isolated SSMR sandbox stack"
	@echo "  ssmr-infra-down       stop the isolated SSMR sandbox stack"
	@echo "  test-ssmr-infra       run isolated SSMR smoke + full-ecosystem PX_E2E marker gate"
	@echo "  test-ssmr-lifecycle   Phase 6: create→dispatch→seal→complete vertical only"
	@echo "  test-ssmr-fiscal      ADR-009 fiscal hard-gate markers (cash/fail/force/shortfall/shift-freeze)"
	@echo "  planning-training-export  PX-PROD-3 baseline export for ML collect-later (Spanner required)"
	@echo "  planning-export-validate  PX-PROD-3 post-export QA (FILE=path [MIN_ROWS=1])"
	@echo "  planning-export-local-cron  PX-LC-2 daily export+validate; appends artifacts/planning-export-audit.log"
	@echo "  planning-export-audit-gate  PX-LC-2 verify N consecutive OK days (DAYS=7 default)"
	@echo "  planning-export-audit-bootstrap  PX-LC-2 local: backfill prior days + cron + gate (DAYS=7)"
	@echo "  fire-drill-ssmr           PX-LC-3 local SSMR fire drills (DRILL=all|A|B|C|D)"
	@echo "  phase0-preflight          PX-PROD-0 gates before terraform (PHASE0_SKIP_WIRE=1 to skip Docker)"
	@echo "  phase0-plan               terraform plan with infra/terraform/staging.tfvars"
	@echo "  phase0-apply              terraform apply with staging.tfvars"
	@echo "  phase0-sync-secrets       sync .env.staging.secrets → GSM (post-apply)"
	@echo "  phase0-migrate            apply Spanner migrations through 20260702 on cloud"
	@echo "  (hybrid)              merge .env.ssmr.hybrid.example for free cloud APIs + local Docker"
	@echo "  ssmr-ecosystem-marker-gate validate PX_E2E log against contracts/ssmr_ecosystem_markers.json"
	@echo "  schema-drift-gate         EH0.3 fail if shop-closed Spanner columns/tables missing"
	@echo "  phase1-gate               Phase-1 exit gate: AR/refund/payout/billing proofs + Soliq + GP simulator"
	@echo "  apply-shop-closed-ddl     EH0.2 apply 20260729 shop-closed DDL + verify (SSMR default)"
	@echo "  load-cert             PX9-F load certification (smoke profile, backend must be up)"
	@echo "  load-cert-ssmr         SSMR stack up + load-cert smoke gate"
	@echo "  load-cert-cloud        load cert against PUBLIC_BASE_URL (staging/prod SLOs)"
	@echo "  cloud-smoke-ssmr       health + client-policy smoke against PUBLIC_BASE_URL"
	@echo "  parity-contract        role-row api-client contract check (PX11-E1)"
	@echo "  parity-contract-full   PX12-A full client path vs backend route check"
	@echo "  gap-hunter-gate        PX12-C event producer/consumer shape gate"
	@echo "  gen-contracts-gate     verify contracts/events.schema.json matches gen-contracts"
	@echo "  partner-openapi-gate   verify contracts/partner.openapi.yaml exists + path markers"
	@echo "  jwt-openapi-gate       verify contracts/jwt-core.openapi.yaml exists + path markers"
	@echo "  px12-preflight         PX12 automated QA gates (contract + gap-hunter + launch-readiness)"
	@echo "  p0-preflight           P0 launch gates (prod profile + k8s + SSMR; see docs/P0_LAUNCH_CHECKLIST.md)"
	@echo "  p1-pilot-weekly        P1 weeks 1–8 weekly gate (see docs/P1_PILOT_CHECKLIST.md)"
	@echo "  wire-ready             Full automated gate before GCP staging wire (see docs/WIRE_READY_STAGING_RUNBOOK.md)"
	@echo "  qa-gate                compile + unit tests (fails if integration tests would be skipped)"
	@echo "  validate-ai-worker-k8s validate ai-worker Kubernetes manifests"
	@echo "  validate-backend-k8s  validate backend-go Kubernetes manifests"
	@echo "  validate-desktop-clients  PX-DESK typecheck + vitest + static export for all Tauri portals"
	@echo "  validate-desktop-updater  PX-DESK-0A/0E pubkey + GCS updater endpoint gate"
	@echo "  validate-launch-readiness validate launch evidence bundle"
	@echo "  docker-build          build backend-go + ai-worker container images (local tag)"
	@echo "  docker-build-backend  build backend-go image only"
	@echo "  docker-build-ai-worker build ai-worker image only"
	@echo "  render-k8s-manifests render infra/k8s into artifacts/k8s-rendered (needs .env.k8s)"
	@echo "  backend-run           run backend-go locally"
	@echo "  backend-build         build all Go binaries"
	@echo "  supplier-portal-dev   run supplier portal Next.js dev server"
	@echo "  marketing-site-dev    run PegasusX marketing site (port 3004)"
	@echo "  terraform-init        initialize terraform working dir"
	@echo "  terraform-plan        preview infra changes"
	@echo "  terraform-apply       apply infra changes"

infra-up:
	docker compose -f infra/docker-compose.yml up -d

infra-down:
	docker compose -f infra/docker-compose.yml down

ssmr-infra-up:
	docker compose -f infra/docker-compose.ssmr.yml up -d

ssmr-infra-down:
	docker compose -f infra/docker-compose.ssmr.yml down

ssmr-infra-reset:
	docker compose -f infra/docker-compose.ssmr.yml down -v --remove-orphans

test-ssmr-infra:
	bash scripts/smoke_ssmr.sh

test-ssmr-lifecycle:
	bash scripts/smoke_ssmr_lifecycle.sh

test-ssmr-fiscal:
	bash scripts/smoke_ssmr_fiscal.sh

planning-training-export:
	bash scripts/planning_training_export.sh

planning-export-validate:
	@test -n "$(FILE)" || (echo "usage: make planning-export-validate FILE=export.jsonl [MIN_ROWS=1]" >&2; exit 1)
	bash scripts/planning_export_validate.sh "$(FILE)" "$(MIN_ROWS)"

planning-export-local-cron:
	bash scripts/planning_export_local_cron.sh

planning-export-audit-gate:
	bash scripts/planning_export_audit_gate.sh "$(DAYS)"

planning-export-audit-bootstrap:
	bash scripts/planning_export_audit_bootstrap.sh "$(DAYS)"

fire-drill-ssmr:
	bash scripts/fire_drill_ssmr.sh "$(DRILL)"

phase0-preflight:
	bash scripts/phase0_preflight.sh

phase0-plan:
	@test -f infra/terraform/staging.tfvars || (echo "copy infra/terraform/staging.tfvars.example to staging.tfvars" >&2; exit 1)
	cd infra/terraform && terraform plan -var-file=staging.tfvars

phase0-apply:
	@test -f infra/terraform/staging.tfvars || (echo "copy infra/terraform/staging.tfvars.example to staging.tfvars" >&2; exit 1)
	cd infra/terraform && terraform apply -var-file=staging.tfvars

phase0-sync-secrets:
	bash scripts/phase0_sync_gsm_secrets.sh

phase0-migrate:
	bash scripts/phase0_apply_spanner_migrations.sh

ssmr-ecosystem-marker-gate:
	@test -n "$(LOG)" || (echo "usage: make ssmr-ecosystem-marker-gate LOG=path/to/ssmr-e2e.log" >&2; exit 1)
	bash scripts/parity/ssmr_ecosystem_marker_gate.sh "$(LOG)"

schema-drift-gate:
	bash scripts/ci_schema_drift_gate.sh

money-path-gate:
	bash scripts/money_path_gate.sh

phase1-gate:
	bash scripts/phase1_gate.sh

apply-shop-closed-ddl:
	bash scripts/apply_shop_closed_ddl.sh

load-cert:
	PATH="/opt/homebrew/bin:/usr/local/bin:$$PATH" bash scripts/load/load_cert.sh

load-cert-ssmr:
	PATH="/opt/homebrew/bin:/usr/local/bin:$$PATH" bash scripts/load/load_cert.sh --with-ssmr

load-cert-cloud:
	bash scripts/load/load_cert_cloud.sh

cloud-smoke-ssmr:
	bash scripts/cloud_smoke_ssmr.sh

parity-contract:
	bash scripts/parity/role_row_contract_check.sh

parity-contract-full:
	bash scripts/parity/role_row_contract_check_full.sh

gap-hunter-gate:
	bash scripts/parity/gap_hunter_gate.sh

gen-contracts-gate:
	bash scripts/parity/gen_contracts_gate.sh

partner-openapi-gate:
	@test -f contracts/partner.openapi.yaml || (echo "missing contracts/partner.openapi.yaml" && exit 1)
	@grep -q '/partner/v1/orders' contracts/partner.openapi.yaml || (echo "partner.openapi.yaml missing /partner/v1/orders" && exit 1)
	@echo "partner-openapi-gate-ok"

jwt-openapi-gate:
	@test -f contracts/jwt-core.openapi.yaml || (echo "missing contracts/jwt-core.openapi.yaml" && exit 1)
	@grep -q '/v1/warehouse/ops/dashboard' contracts/jwt-core.openapi.yaml || (echo "jwt-core.openapi.yaml missing /v1/warehouse/ops/dashboard" && exit 1)
	@grep -q '/v1/supplier/inventory' contracts/jwt-core.openapi.yaml || (echo "jwt-core.openapi.yaml missing /v1/supplier/inventory" && exit 1)
	@grep -q '/v1/order' contracts/jwt-core.openapi.yaml || (echo "jwt-core.openapi.yaml missing /v1/order" && exit 1)
	@grep -q '/v1/payloader/manifests' contracts/jwt-core.openapi.yaml || (echo "jwt-core.openapi.yaml missing /v1/payloader/manifests" && exit 1)
	@echo "jwt-openapi-gate-ok"

px12-preflight:
	bash scripts/qa/px12_preflight.sh

p0-preflight:
	bash scripts/p0_launch_preflight.sh

p1-pilot-weekly:
	bash scripts/p1_pilot_weekly.sh

wire-ready:
	bash scripts/wire_ready.sh

validate-production-profile:
	bash scripts/validate_production_profile.sh

qa-gate: backend-build
	@echo "==> qa-gate: unit tests (PARITY_REQUIRE_SPANNER=1 — integration tests must not skip)"
	cd apps/backend-go && PARITY_REQUIRE_SPANNER=1 go test ./... -count=1 -short

validate-ai-worker-k8s:
	bash scripts/validate_ai_worker_k8s.sh

validate-backend-k8s:
	bash scripts/validate_backend_k8s.sh

validate-desktop-clients:
	bash scripts/validate_desktop_clients.sh

validate-desktop-updater:
	bash scripts/validate_desktop_updater.sh

validate-launch-readiness:
	python3 scripts/validate_launch_readiness.py

docker-build:
	bash scripts/docker_build.sh all

docker-build-backend:
	bash scripts/docker_build.sh backend

docker-build-ai-worker:
	bash scripts/docker_build.sh ai-worker

render-k8s-manifests:
	bash scripts/render_k8s_manifests.sh

render-k8s-from-terraform:
	bash scripts/render_k8s_from_terraform.sh

backend-run:
	cd apps/backend-go && go run ./...

backend-build:
	for dir in apps/ai-worker apps/backend-go packages/config packages/optimizer-contract; do (cd $$dir && go build ./...); done

supplier-portal-dev:
	cd apps/supplier-portal && pnpm dev

marketing-site-dev:
	cd apps/marketing-site && pnpm dev

terraform-init:
	cd infra/terraform && terraform init

terraform-plan:
	cd infra/terraform && terraform plan

terraform-apply:
	cd infra/terraform && terraform apply

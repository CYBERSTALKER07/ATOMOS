.PHONY: help infra-up infra-down ssmr-infra-up ssmr-infra-down test-ssmr-infra ssmr-ecosystem-marker-gate planning-training-export load-cert load-cert-ssmr load-cert-cloud cloud-smoke-ssmr parity-contract parity-contract-full gap-hunter-gate gen-contracts-gate px12-preflight p0-preflight p1-pilot-weekly wire-ready qa-gate validate-ai-worker-k8s validate-backend-k8s validate-launch-readiness validate-production-profile docker-build docker-build-backend docker-build-ai-worker backend-run backend-build supplier-portal-dev marketing-site-dev

help:
	@echo "pegasusX make targets"
	@echo "  infra-up              start Spanner emulator, Redis, Kafka via docker-compose"
	@echo "  infra-down            stop infra"
	@echo "  ssmr-infra-up         start the isolated SSMR sandbox stack"
	@echo "  ssmr-infra-down       stop the isolated SSMR sandbox stack"
	@echo "  test-ssmr-infra       run isolated SSMR smoke + full-ecosystem PX_E2E marker gate"
	@echo "  planning-training-export  PX-PROD-3 baseline export for ML collect-later (Spanner required)"
	@echo "  (hybrid)              merge .env.ssmr.hybrid.example for free cloud APIs + local Docker"
	@echo "  ssmr-ecosystem-marker-gate validate PX_E2E log against contracts/ssmr_ecosystem_markers.json"
	@echo "  load-cert             PX9-F load certification (smoke profile, backend must be up)"
	@echo "  load-cert-ssmr         SSMR stack up + load-cert smoke gate"
	@echo "  load-cert-cloud        load cert against PUBLIC_BASE_URL (staging/prod SLOs)"
	@echo "  cloud-smoke-ssmr       health + client-policy smoke against PUBLIC_BASE_URL"
	@echo "  parity-contract        role-row api-client contract check (PX11-E1)"
	@echo "  parity-contract-full   PX12-A full client path vs backend route check"
	@echo "  gap-hunter-gate        PX12-C event producer/consumer shape gate"
	@echo "  gen-contracts-gate     verify contracts/events.schema.json matches gen-contracts"
	@echo "  px12-preflight         PX12 automated QA gates (contract + gap-hunter + launch-readiness)"
	@echo "  p0-preflight           P0 launch gates (prod profile + k8s + SSMR; see docs/P0_LAUNCH_CHECKLIST.md)"
	@echo "  p1-pilot-weekly        P1 weeks 1–8 weekly gate (see docs/P1_PILOT_CHECKLIST.md)"
	@echo "  wire-ready             Full automated gate before GCP staging wire (see docs/WIRE_READY_STAGING_RUNBOOK.md)"
	@echo "  qa-gate                compile + unit tests (fails if integration tests would be skipped)"
	@echo "  validate-ai-worker-k8s validate ai-worker Kubernetes manifests"
	@echo "  validate-backend-k8s  validate backend-go Kubernetes manifests"
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

planning-training-export:
	bash scripts/planning_training_export.sh

ssmr-ecosystem-marker-gate:
	@test -n "$(LOG)" || (echo "usage: make ssmr-ecosystem-marker-gate LOG=path/to/ssmr-e2e.log" >&2; exit 1)
	bash scripts/parity/ssmr_ecosystem_marker_gate.sh "$(LOG)"

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

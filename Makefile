.PHONY: help verify-boundary-isolation pegasusx-verify pegasusx-backend-build pegasusx-backend-test pegasusx-typecheck pegasusx-test pegasusx-launch-readiness pegasusx-ssmr-up pegasusx-ssmr-down pegasusx-ssmr-smoke antimatter-dev antimatter-build pegasus-backend-build split-repos

help:
	@echo "V.O.I.D Enterprise 3-Way Ecosystem Orchestrator"
	@echo ""
	@echo "  1. pegasusX (Priority-1 Prod: SSMR B2B Distribution OS — 1 Supplier, 1,000s of Retailers)"
	@echo "    make pegasusx-verify            Run full fast verification (Go build/tests, TS typecheck/tests, Spanner & Launch gates)"
	@echo "    make pegasusx-backend-build     Compile all 7 Go modules in pegasusX/go.work"
	@echo "    make pegasusx-backend-test      Run Go unit & race-detector tests across hot paths"
	@echo "    make pegasusx-typecheck         Typecheck all 21 PNPM/Turborepo packages & portals"
	@echo "    make pegasusx-test              Run all Vitest suites across pegasusX packages & portals"
	@echo "    make pegasusx-launch-readiness  Run launch readiness + parity + K8s + Spanner stale-read gates"
	@echo "    make pegasusx-ssmr-up           Boot local 10-container SSMR Docker Compose stack"
	@echo "    make pegasusx-ssmr-down         Stop local SSMR Docker Compose stack"
	@echo "    make pegasusx-ssmr-smoke        Run end-to-end SSMR smoke certification"
	@echo ""
	@echo "  2. antimatter (Public Marketing, 3D Showcase, xAI Assistant & Supabase Lead Hub — Netlify)"
	@echo "    make antimatter-dev             Run Antimatter Next.js 16 dev server"
	@echo "    make antimatter-build           Build Antimatter Next.js 16 production bundle"
	@echo ""
	@echo "  3. pegasus (Priority-2 R&D: Multi-Supplier Marketplace + Spatial 3D / Rust Engine)"
	@echo "    make pegasus-backend-build      Compile pegasus Go backend"
	@echo ""
	@echo "  Architecture & 3-Way Repository Separation"
	@echo "    make verify-boundary-isolation  Verify zero cross-project imports & standalone manifest readiness across all 3 codebases"
	@echo "    make split-repos                Extract standalone pegasusx, antimatter, and pegasus git repositories via git subtree"

verify-boundary-isolation:
	bash scripts/split-repositories.sh --verify-only

pegasusx-backend-build:
	$(MAKE) -C pegasusX backend-build

pegasusx-backend-test:
	cd pegasusX/apps/backend-go && go test -race ./outbox/... ./catalog/... ./promotion/... ./stocklots/...

pegasusx-typecheck:
	cd pegasusX && pnpm run typecheck

pegasusx-test:
	cd pegasusX && pnpm run test

pegasusx-launch-readiness:
	cd pegasusX && bash scripts/validate_spanner_stale_reads.sh && python3 scripts/validate_launch_readiness.py

pegasusx-verify: verify-boundary-isolation pegasusx-backend-build pegasusx-backend-test pegasusx-typecheck pegasusx-test pegasusx-launch-readiness
	@echo "=== pegasusX production verification complete: ALL GATES GREEN ==="

pegasusx-ssmr-up:
	$(MAKE) -C pegasusX ssmr-infra-up

pegasusx-ssmr-down:
	$(MAKE) -C pegasusX ssmr-infra-down

pegasusx-ssmr-smoke:
	$(MAKE) -C pegasusX test-ssmr-infra

antimatter-dev:
	cd pegasusX/antimater && npm run dev

antimatter-build:
	cd pegasusX/antimater && npm run build

pegasus-backend-build:
	cd pegasus/apps/backend-go && go build ./...

split-repos:
	bash scripts/split-repositories.sh --execute

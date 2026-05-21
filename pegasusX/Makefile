.PHONY: help infra-up infra-down ssmr-infra-up ssmr-infra-down test-ssmr-infra backend-run backend-build supplier-portal-dev

help:
	@echo "pegasusX make targets"
	@echo "  infra-up              start Spanner emulator, Redis, Kafka via docker-compose"
	@echo "  infra-down            stop infra"
	@echo "  ssmr-infra-up         start the isolated SSMR sandbox stack"
	@echo "  ssmr-infra-down       stop the isolated SSMR sandbox stack"
	@echo "  test-ssmr-infra       run the isolated SSMR smoke-check and always tear down"
	@echo "  backend-run           run backend-go locally"
	@echo "  backend-build         build all Go binaries"
	@echo "  supplier-portal-dev   run supplier portal Next.js dev server"
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

test-ssmr-infra:
	bash scripts/smoke_ssmr.sh

backend-run:
	cd apps/backend-go && go run ./...

backend-build:
	go build ./...

supplier-portal-dev:
	cd apps/supplier-portal && pnpm dev

terraform-init:
	cd infra/terraform && terraform init

terraform-plan:
	cd infra/terraform && terraform plan

terraform-apply:
	cd infra/terraform && terraform apply

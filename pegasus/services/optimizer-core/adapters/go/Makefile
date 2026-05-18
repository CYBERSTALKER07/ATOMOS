.PHONY: run
run:
	go run ./cmd/optimizer-worker

.PHONY: gen-proto
gen-proto:
	@echo "Generate Go stubs from ../../proto/optimizer_core.proto using protoc + protoc-gen-go + protoc-gen-go-grpc."
	@echo "Wire generated types into internal/optimizergrpc/client.go transport methods."

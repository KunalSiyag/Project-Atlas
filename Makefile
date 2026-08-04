# Project Atlas — common local developer commands
# Requires Go at /usr/local/go/bin or on PATH.

export PATH := /usr/local/go/bin:$(PATH)

GATEWAY_DIR := services/inference-gateway
ORCH_DIR    := services/agent-orchestrator
CLI_DIR     := clients/atlas-cli
GATEWAY_URL ?= http://127.0.0.1:8080
GATEWAY_ADDR ?= 127.0.0.1:8080
ORCH_ADDR   ?= 127.0.0.1:8090

.PHONY: help gateway-test cli-test orch-test test gateway-run orch-run cli-build cli-run cli-health cli-generate compose-fake smoke

help:
	@echo "Atlas make targets"
	@echo "  make gateway-test   run inference-gateway unit tests"
	@echo "  make orch-test      run agent-orchestrator unit tests"
	@echo "  make cli-test       run atlas-cli unit tests"
	@echo "  make test           run all Go test suites"
	@echo "  make gateway-run    start gateway with fake provider (foreground)"
	@echo "  make orch-run       start agent-orchestrator (foreground)"
	@echo "  make cli-build      build ./clients/atlas-cli/atlas binary"
	@echo "  make cli-run        interactive CLI chat"
	@echo "  make cli-health     CLI health against GATEWAY_URL"
	@echo "  make cli-generate MSG='hi'  one-shot generate"
	@echo "  make smoke          one-shot fake gateway + CLI health/generate/openapi"
	@echo "  make compose-fake   docker compose with fake provider + postgres"

gateway-test:
	cd $(GATEWAY_DIR) && go test ./...

orch-test:
	cd $(ORCH_DIR) && go test ./...

cli-test:
	cd $(CLI_DIR) && go test ./...

test: gateway-test orch-test cli-test

gateway-run:
	cd $(GATEWAY_DIR) && \
		ATLAS_INFERENCE_ADDR=$(GATEWAY_ADDR) \
		ATLAS_INFERENCE_PROVIDER=fake \
		ATLAS_INFERENCE_MODEL=fake-model \
		go run ./cmd/inference-gateway

orch-run:
	cd $(ORCH_DIR) && \
		ATLAS_ORCHESTRATOR_ADDR=$(ORCH_ADDR) \
		ATLAS_INFERENCE_GATEWAY_URL=$(GATEWAY_URL) \
		go run ./cmd/agent-orchestrator

cli-build:
	cd $(CLI_DIR) && go build -o atlas .

cli-run: cli-build
	cd $(CLI_DIR) && ./atlas -url $(GATEWAY_URL) -model fake-model

cli-health: cli-build
	cd $(CLI_DIR) && ./atlas -url $(GATEWAY_URL) health

cli-generate: cli-build
	cd $(CLI_DIR) && ./atlas -url $(GATEWAY_URL) -no-color generate "$(MSG)"

compose-fake:
	cd deploy && docker compose -f docker-compose.yml -f docker-compose.fake.yml up --build

smoke:
	./scripts/dev_smoke.sh

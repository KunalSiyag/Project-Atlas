# Project Atlas — common local developer commands
# Requires Go at /usr/local/go/bin or on PATH.

export PATH := /usr/local/go/bin:$(PATH)

GATEWAY_DIR := services/inference-gateway
ORCH_DIR    := services/agent-orchestrator
CLI_DIR     := clients/atlas-cli
GATEWAY_URL ?= http://127.0.0.1:8080
GATEWAY_ADDR ?= 127.0.0.1:8080
ORCH_ADDR   ?= 127.0.0.1:8090

.PHONY: help gateway-test cli-test orch-test eval-test eval eval-gate bench test gateway-run orch-run cli-build cli-run cli-health cli-generate compose-fake compose-otel smoke

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
	@echo "  make compose-otel   compose config for OTEL+Grafana overlay"
	@echo "  make eval           run atlas-eval on sample fixtures"
	@echo "  make eval-gate      CI-style eval gate (fails if fixtures fail)"
	@echo "  make bench          load/bench against GATEWAY_URL/healthz (JSON metrics)"

gateway-test:
	cd $(GATEWAY_DIR) && go test ./...

orch-test:
	cd $(ORCH_DIR) && go test ./...

cli-test:
	cd $(CLI_DIR) && go test ./...

test: gateway-test orch-test cli-test eval-test bench-test

eval-test:
	cd tools/atlas-eval && go test ./...

bench-test:
	cd tools/atlas-bench && go test ./...

# CI-style gate: run atlas-eval harness tests against in-repo fixtures; non-zero exit on failure.
eval-gate:
	@echo "eval-gate: running tools/atlas-eval go test (fixtures must pass)"
	cd tools/atlas-eval && go test ./... -count=1 -timeout 120s
	@echo "eval-gate: PASS"

eval:
	cd tools/atlas-eval && go run . -dataset ../../fixtures/eval/v1/sample.json -out ../../eval-report.json

# Load/bench: hit a real service path; writes machine-readable latency/throughput JSON.
# Default target is GATEWAY_URL/healthz. Override: make bench URL=http://127.0.0.1:8090 PATH=/healthz
bench:
	cd tools/atlas-bench && go run . -url $(GATEWAY_URL) -path /healthz -n 40 -c 8 -out ../../bench-metrics.json

compose-otel:
	cd deploy && docker compose -f docker-compose.yml -f docker-compose.otel.yml config >/dev/null && echo "compose otel: OK"

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

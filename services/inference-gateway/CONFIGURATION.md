# Inference Gateway Configuration

## Purpose

Document process configuration for local development and future deployment.

## Configuration keys

| Variable | Default | Meaning |
|---|---|---|
| `ATLAS_INFERENCE_ADDR` | `:8080` | HTTP listen address |
| `ATLAS_INFERENCE_PROVIDER` | `fake` | Provider adapter name |
| `ATLAS_INFERENCE_MODEL` | `fake-model` | Default model sent to the selected provider |
| `ATLAS_OLLAMA_URL` | `http://localhost:11434` | Ollama server base URL |
| `ATLAS_INFERENCE_MAX_ATTEMPTS` | `3` | Maximum provider attempts, including the first call |
| `ATLAS_DATABASE_URL` | empty | PostgreSQL connection string for durable inference runs |

The current implementation supports the deterministic `fake` provider. A real provider adapter can use the same configuration key after its implementation is added.

Routing is currently an in-process static map: `plan` uses `fake-planner`, `act` uses `fake-model`, and unknown task classes use the default route.

## Example

```bash
ATLAS_INFERENCE_ADDR=127.0.0.1:9090 \
ATLAS_INFERENCE_PROVIDER=ollama \
ATLAS_INFERENCE_MODEL=gemma3 \
ATLAS_OLLAMA_URL=http://localhost:11434 \
go run ./cmd/inference-gateway
```

The default `fake` provider requires no external service. To use Ollama, install and run Ollama locally, make sure the selected model is available, and set `ATLAS_INFERENCE_PROVIDER=ollama`. Ollama’s default local API base URL is `http://localhost:11434/api`.

Secrets are not configured in this first slice. When hosted providers are added, credentials must come from environment variables or a secret manager and must never be committed to the repository.

Provider errors are retried only when classified as transient (`rate_limit`, `timeout`, or `unavailable`). Invalid requests and authentication failures return immediately.

## Telemetry

The service uses the OpenTelemetry Go API with instrumentation name `github.com/KunalSiyag/Project-Atlas/services/inference-gateway`. It emits request and provider-attempt spans plus request, failure, and latency instruments. The current module does not force an exporter or SDK; deployment can install/configure an OTLP or console exporter through the application’s global OpenTelemetry providers.

# Local Atlas Data Plane

## Purpose

Provide the local development dependencies used by Atlas services.

## Scope

This Compose stack builds the inference gateway and runs it with PostgreSQL, Redis, Qdrant, MinIO, and Ollama. Other Atlas application services are not included yet because their container entrypoints and migrations are not implemented.

## Start

```bash
cd deploy
cp .env.example .env
docker compose up -d
docker compose ps
```

Stop the services without deleting data:

```bash
docker compose down
```

To remove the named development volumes as well, use `docker compose down -v`. This deletes local database, vector, object, and model data.

## Endpoints

| Service | Host endpoint | Purpose |
|---|---|---|
| Inference gateway | `localhost:8080` | Chat inference API |
| PostgreSQL | `localhost:5432` | Relational state |
| Redis | `localhost:6379` | Cache, rate limits, short-term state |
| Qdrant HTTP | `localhost:6333` | Vector search |
| Qdrant gRPC | `localhost:6334` | Vector client path |
| MinIO API | `localhost:9000` | S3-compatible objects |
| MinIO console | `localhost:9001` | Local object-store UI |
| Ollama | `localhost:11434` | Local model API |

Inside the Compose network, services use hostnames such as `postgres`, `redis`, `qdrant`, `minio`, and `ollama` rather than `localhost`. The gateway is configured with `ATLAS_OLLAMA_URL=http://ollama:11434` for this reason.

The gateway also uses PostgreSQL through `ATLAS_DATABASE_URL` and automatically applies the `inference_runs` and `conversations` schemas on startup. Each request returns an `X-Run-ID` that can be used to inspect its persisted lifecycle record.

Inspect a run:

```bash
curl http://localhost:8080/v1/runs/<run-id>
```

Create and inspect a conversation:

```bash
curl -X POST http://localhost:8080/v1/conversations \
  -H 'Content-Type: application/json' \
  -d '{"title":"Local experiment"}'
curl http://localhost:8080/v1/conversations/<conversation-id>
curl -X POST http://localhost:8080/v1/conversations/<conversation-id>/messages \
  -H 'Content-Type: application/json' \
  -d '{"content":"Summarize our discussion"}'
```

## Notes

- Images use development-oriented defaults; pin and scan images before production use.
- Credentials in `.env.example` are intentionally weak local defaults and must be replaced outside a personal workstation.
- Ollama does not automatically download a model. After startup, pull one with `docker compose exec ollama ollama pull gemma3` if the image supports that command.
- The gateway image is built from `services/inference-gateway/Dockerfile` and runs as a non-root user.
- The named volumes are persistent. Backups and restore procedures are not yet implemented.

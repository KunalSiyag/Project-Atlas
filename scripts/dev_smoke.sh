#!/usr/bin/env bash
# One-command local smoke: fake gateway + atlas-cli health/generate/openapi.
# Usage: from repo root → ./scripts/dev_smoke.sh
set -euo pipefail

export PATH="/usr/local/go/bin:${PATH}"
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
ADDR="${ATLAS_SMOKE_ADDR:-127.0.0.1:18082}"
URL="http://${ADDR}"
TMP="${TMPDIR:-/tmp}/atlas-smoke-$$"
mkdir -p "$TMP"
cleanup() {
  if [[ -f "$TMP/gateway.pid" ]]; then
    kill "$(cat "$TMP/gateway.pid")" 2>/dev/null || true
    wait "$(cat "$TMP/gateway.pid")" 2>/dev/null || true
  fi
  rm -rf "$TMP"
}
trap cleanup EXIT

echo "== building =="
(cd "$ROOT/services/inference-gateway" && go build -o "$TMP/inference-gateway" ./cmd/inference-gateway)
(cd "$ROOT/clients/atlas-cli" && go build -o "$TMP/atlas" .)

echo "== starting gateway (fake) on ${ADDR} =="
ATLAS_INFERENCE_ADDR="$ADDR" \
ATLAS_INFERENCE_PROVIDER=fake \
ATLAS_INFERENCE_MODEL=fake-model \
"$TMP/inference-gateway" >"$TMP/gateway.log" 2>&1 &
echo $! >"$TMP/gateway.pid"

for _ in $(seq 1 50); do
  if curl -sf "${URL}/healthz" >/dev/null; then
    break
  fi
  sleep 0.1
done
curl -sf "${URL}/healthz" >/dev/null

echo "== atlas health =="
"$TMP/atlas" -url "$URL" health

echo "== atlas generate =="
out="$("$TMP/atlas" -url "$URL" -no-color generate "smoke hello")"
echo "$out"
echo "$out" | grep -q "fake response to: smoke hello"
echo "$out" | grep -q "█████╗"

echo "== openapi =="
spec="$("$TMP/atlas" -url "$URL" openapi)"
echo "$spec" | head -n 5
echo "$spec" | grep -q "Atlas Inference Gateway"
echo "$spec" | grep -q "/v1/generate"

echo "== OK: dev smoke passed =="

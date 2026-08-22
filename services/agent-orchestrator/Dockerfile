FROM golang:1.25-alpine AS builder

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/agent-orchestrator ./cmd/agent-orchestrator

FROM alpine:3.21

RUN adduser -D -H -u 10001 atlas
COPY --from=builder /out/agent-orchestrator /usr/local/bin/agent-orchestrator
# Optional prompt pack mount path (repo prompts/ bind-mounted in Compose).
RUN mkdir -p /prompts && chown atlas:atlas /prompts

USER atlas
EXPOSE 8090
ENTRYPOINT ["/usr/local/bin/agent-orchestrator"]

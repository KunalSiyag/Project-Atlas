FROM golang:1.25-alpine AS builder

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/inference-gateway ./cmd/inference-gateway

FROM alpine:3.21

RUN adduser -D -H -u 10001 atlas
COPY --from=builder /out/inference-gateway /usr/local/bin/inference-gateway

USER atlas
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/inference-gateway"]

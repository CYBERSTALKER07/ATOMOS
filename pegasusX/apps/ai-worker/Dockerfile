# Build context MUST be the pegasusX module root (directory containing go.work).
FROM golang:1.25-bookworm AS builder

RUN apt-get update && apt-get install -y --no-install-recommends gcc libc6-dev git ca-certificates \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /src

COPY go.work go.work.sum ./
COPY packages/config/go.mod ./packages/config/
COPY apps/backend-go/go.mod apps/backend-go/go.sum ./apps/backend-go/
COPY apps/ai-worker/go.mod apps/ai-worker/go.sum ./apps/ai-worker/
COPY apps/handoff-service/go.mod apps/handoff-service/go.sum ./apps/handoff-service/
COPY packages/handoff/go.mod packages/handoff/go.sum ./packages/handoff/
COPY packages/optimizer-contract/go.mod ./packages/optimizer-contract/

WORKDIR /src/apps/ai-worker
RUN go mod download

WORKDIR /src
COPY packages/config ./packages/config
COPY packages/handoff ./packages/handoff
COPY packages/optimizer-contract ./packages/optimizer-contract
COPY apps/handoff-service ./apps/handoff-service
COPY apps/backend-go ./apps/backend-go
COPY apps/ai-worker ./apps/ai-worker

RUN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s" \
    -o /pegasusx-ai-worker \
    ./apps/ai-worker

FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates tzdata && rm -rf /var/lib/apt/lists/*

COPY --from=builder /pegasusx-ai-worker /usr/local/bin/pegasusx-ai-worker

EXPOSE 8081

USER nobody

ENTRYPOINT ["/usr/local/bin/pegasusx-ai-worker"]

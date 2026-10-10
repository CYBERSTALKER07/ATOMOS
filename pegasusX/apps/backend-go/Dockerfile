# Build context MUST be the pegasusX module root (directory containing go.work).
# h3-go requires CGO — use bookworm builder, not alpine.
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

WORKDIR /src/apps/backend-go
RUN go mod download

WORKDIR /src
COPY packages/config ./packages/config
COPY packages/handoff ./packages/handoff
COPY packages/optimizer-contract ./packages/optimizer-contract
COPY apps/handoff-service ./apps/handoff-service
COPY apps/backend-go ./apps/backend-go

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s" \
    -o /pegasusx-backend \
    ./apps/backend-go

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s" \
    -o /pegasusx-setup \
    ./apps/backend-go/cmd/setup

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s" \
    -o /planning-training-export \
    ./apps/backend-go/cmd/planning-training-export

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s" \
    -o /planning-accuracy \
    ./apps/backend-go/cmd/planning-accuracy

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s" \
    -o /planning-forecast \
    ./apps/backend-go/cmd/planning-forecast

# Runtime must be glibc-based: h3-go forces CGO_ENABLED=1, so the binary links
# against bookworm libc. Alpine (musl) yields "exec ...: no such file or directory".
FROM debian:bookworm-slim

LABEL org.opencontainers.image.title="PegasusX Backend" \
      org.opencontainers.image.description="Core backend service for PegasusX logistics & commerce" \
      org.opencontainers.image.source="https://github.com/pegasusx/pegasusx"

RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates tzdata curl \
    && rm -rf /var/lib/apt/lists/* \
    && groupadd -g 10001 pegasus \
    && useradd -u 10001 -g pegasus -d /runtime -s /bin/false pegasus

WORKDIR /runtime
COPY --from=builder /pegasusx-backend /usr/local/bin/pegasusx-backend
COPY --from=builder /pegasusx-setup /usr/local/bin/pegasusx-setup
COPY --from=builder /planning-training-export /usr/local/bin/planning-training-export
COPY --from=builder /planning-accuracy /usr/local/bin/planning-accuracy
COPY --from=builder /planning-forecast /usr/local/bin/planning-forecast
COPY apps/backend-go/schema ./apps/backend-go/schema

RUN chown -R pegasus:pegasus /runtime

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=5s --start-period=20s --retries=3 \
  CMD curl -f http://localhost:8080/healthz || exit 1

USER pegasus:pegasus

ENTRYPOINT ["/usr/local/bin/pegasusx-backend"]


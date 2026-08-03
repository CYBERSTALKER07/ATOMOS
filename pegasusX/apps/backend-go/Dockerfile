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

RUN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s" \
    -o /pegasusx-backend \
    ./apps/backend-go

RUN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s" \
    -o /pegasusx-setup \
    ./apps/backend-go/cmd/setup

RUN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s" \
    -o /planning-training-export \
    ./apps/backend-go/cmd/planning-training-export

# Runtime must be glibc-based: h3-go forces CGO_ENABLED=1, so the binary links
# against bookworm libc. Alpine (musl) yields "exec ...: no such file or directory".
FROM debian:bookworm-slim

RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates tzdata \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /runtime
COPY --from=builder /pegasusx-backend /usr/local/bin/pegasusx-backend
COPY --from=builder /pegasusx-setup /usr/local/bin/pegasusx-setup
COPY --from=builder /planning-training-export /usr/local/bin/planning-training-export
COPY apps/backend-go/schema ./apps/backend-go/schema

EXPOSE 8080

USER nobody

ENTRYPOINT ["/usr/local/bin/pegasusx-backend"]

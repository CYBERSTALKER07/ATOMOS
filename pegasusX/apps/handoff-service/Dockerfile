FROM golang:1.25-bookworm AS build
WORKDIR /src
COPY go.work go.work.sum ./
COPY sdk/partner/go/go.mod ./sdk/partner/go/
COPY packages/config/go.mod ./packages/config/
COPY apps/backend-go/go.mod apps/backend-go/go.sum ./apps/backend-go/
COPY apps/ai-worker/go.mod apps/ai-worker/go.sum ./apps/ai-worker/
COPY apps/handoff-service/go.mod apps/handoff-service/go.sum ./apps/handoff-service/
COPY packages/handoff/go.mod packages/handoff/go.sum ./packages/handoff/
COPY packages/optimizer-contract/go.mod ./packages/optimizer-contract/
COPY packages/handoff ./packages/handoff
COPY apps/handoff-service ./apps/handoff-service
WORKDIR /src/apps/handoff-service
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /handoff-service .

FROM gcr.io/distroless/base-debian12:nonroot

LABEL org.opencontainers.image.title="PegasusX Handoff Service" \
      org.opencontainers.image.description="Decoupled pickup and delivery handoff token verification service" \
      org.opencontainers.image.source="https://github.com/pegasusx/pegasusx"

COPY --from=build /handoff-service /handoff-service
ENV HTTP_PORT=8082
EXPOSE 8082
USER nonroot:nonroot
ENTRYPOINT ["/handoff-service"]


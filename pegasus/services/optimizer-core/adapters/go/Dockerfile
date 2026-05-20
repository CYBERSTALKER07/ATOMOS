FROM golang:1.25-bookworm AS builder

WORKDIR /src/services/optimizer-core/adapters/go

COPY services/optimizer-core/adapters/go/go.mod services/optimizer-core/adapters/go/go.sum ./
COPY packages/optimizer-contract /src/packages/optimizer-contract
RUN go mod edit -replace optimizercontract=/src/packages/optimizer-contract
RUN go mod download

COPY services/optimizer-core/adapters/go/cmd ./cmd
COPY services/optimizer-core/adapters/go/internal ./internal
RUN go mod tidy

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/optimizer-worker ./cmd/optimizer-worker

FROM gcr.io/distroless/base-debian12

COPY --from=builder /out/optimizer-worker /usr/local/bin/optimizer-worker

ENTRYPOINT ["/usr/local/bin/optimizer-worker"]

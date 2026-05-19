FROM golang:1.25-bookworm AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY cmd ./cmd
COPY internal ./internal

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/optimizer-worker ./cmd/optimizer-worker

FROM gcr.io/distroless/base-debian12

COPY --from=builder /out/optimizer-worker /usr/local/bin/optimizer-worker

ENTRYPOINT ["/usr/local/bin/optimizer-worker"]

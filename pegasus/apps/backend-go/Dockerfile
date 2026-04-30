# Stage 1: The Builder Matrix
FROM golang:1.25-alpine AS builder
WORKDIR /app

# Copy the entire monorepo context to resolve local package imports
COPY . .

# Build the Go binary (CGO disabled for pure static compilation)
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o /lab-gateway ./apps/backend-go/main.go

# Stage 2: The Execution Environment
FROM alpine:latest
# Install root certificates so Go can make HTTPS calls to Payme/Click and Google Maps
RUN apk --no-cache add ca-certificates tzdata

WORKDIR /root/
COPY --from=builder /lab-gateway /usr/local/bin/lab-gateway

# Execute
EXPOSE 8080
ENTRYPOINT ["lab-gateway"]

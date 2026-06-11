FROM golang:1.25-bookworm AS build
WORKDIR /src
COPY go.work go.work.sum ./
COPY packages/handoff ./packages/handoff
COPY apps/handoff-service ./apps/handoff-service
WORKDIR /src/apps/handoff-service
RUN go build -o /handoff-service .

FROM gcr.io/distroless/base-debian12
COPY --from=build /handoff-service /handoff-service
ENV HTTP_PORT=8082
EXPOSE 8082
ENTRYPOINT ["/handoff-service"]

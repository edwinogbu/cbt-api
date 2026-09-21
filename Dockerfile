# Builds the same cbt-api binary that runs in the cloud, unmodified -
# a School CBT Node is this image pointed at a local Postgres/Redis
# instead of the cloud ones, per the offline-first plan's "same binary,
# two deployment locations" requirement (config is already fully
# env-var-driven, no code branches on deployment target).

FROM golang:1.25-alpine AS build
WORKDIR /src

RUN apk add --no-cache git

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/cbt-api ./cmd/server

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata && \
    adduser -D -u 10001 cbtapi
WORKDIR /app
COPY --from=build /out/cbt-api /app/cbt-api

USER cbtapi
EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=5s --start-period=15s --retries=3 \
    CMD wget -qO- http://localhost:8080/api/v1/health || exit 1

ENTRYPOINT ["/app/cbt-api"]

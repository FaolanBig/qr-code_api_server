FROM golang:1.24-alpine AS build

WORKDIR /src
COPY go.mod go.sum ./
COPY vendor ./vendor
COPY cmd ./cmd
COPY internal ./internal
RUN CGO_ENABLED=0 go build -mod=vendor -trimpath -ldflags="-s -w" -o /out/qrcode-api ./cmd/server

FROM debian:12-slim

RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates curl && \
    rm -rf /var/lib/apt/lists/* && \
    groupadd --system app && useradd --system --gid app --create-home app

WORKDIR /app
COPY --from=build --chown=app:app /out/qrcode-api /app/qrcode-api
COPY --chown=app:app assets /app/assets
COPY --chown=app:app docs/openapi.yaml /app/docs/openapi.yaml

USER app
EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD curl --fail --silent --show-error http://127.0.0.1:8080/healthz >/dev/null || exit 1

ENTRYPOINT ["/app/qrcode-api"]

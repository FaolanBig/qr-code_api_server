# QR Code API Server

A production-oriented HTTP service written in Go that converts structured information into QR codes. The API is versioned at `/api/v1`, protected by an API key, and returns PNG QR codes directly in JSON as both Base64 and a data URI.

## Features

- URL QR codes
- vCard contact QR codes
- Wi-Fi configuration QR codes (WPA, WEP, and open networks)
- iCalendar event QR codes
- Plain text QR codes
- Small-file payload QR codes for embeddable data URIs
- Number QR codes
- Email, SMS, and geolocation QR codes as useful additional types
- Optional custom PNG logos or a server-configured fallback logo at the QR center
- API-key authentication via `X-API-Key` or `Authorization: Bearer`
- JSON validation, structured errors, request logging, security headers, and graceful shutdown
- Kubernetes Deployment, Service, ConfigMap, and Secret example
- Small, non-root container with a basic shell and `curl`
- Public OpenAPI 3.1 specification at `/openapi.yaml`

## Quick start

### Local Go

```bash
export API_KEY="replace-with-at-least-16-random-characters"
go run ./cmd/server
```

Call the health endpoint:

```bash
curl http://localhost:8080/healthz
```

Create a URL QR code:

```bash
curl --fail --request POST http://localhost:8080/api/v1/qr/url \
  --header 'Content-Type: application/json' \
  --header 'X-API-Key: replace-with-at-least-16-random-characters' \
  --data '{"url":"https://example.com","size":256,"recovery":"medium"}' \
  | jq -r '.qr_code.data_uri' \
  | sed 's/^data:image\/png;base64,//' \
  | base64 --decode > qr.png
```

### Docker

```bash
docker build -t qr-code-api:local .
docker run --rm --name qr-code-api \
  --publish 8080:8080 \
  --env API_KEY="replace-with-at-least-16-random-characters" \
  qr-code-api:local

curl --fail http://localhost:8080/healthz
```

### Docker Compose

Copy `.env.example` to `.env`, replace the placeholder API key, and start the service:

```bash
cp .env.example .env
# Edit .env and replace API_KEY=replace-with-a-random-secret
docker compose up --build -d
docker compose ps
curl --fail http://127.0.0.1:8080/healthz
```

Stop and remove the service with:

```bash
docker compose down
```

The Compose configuration builds the same Dockerfile and runs the container read-only with dropped capabilities. The runtime uses Debian 12 slim, while the build stage uses Go Alpine; the container runs as a non-root user, includes `/bin/sh` and `curl`, and has a built-in Docker health check.

## API overview

Authentication applies to every `/api/v1/*` endpoint. The public endpoints are `/healthz`, `/readyz`, and `/openapi.yaml`; `/openapi.yaml` contains the complete machine-readable API specification.

```text
POST /api/v1/qr/url
POST /api/v1/qr/vcard
POST /api/v1/qr/wifi
POST /api/v1/qr/calendar
POST /api/v1/qr/text
POST /api/v1/qr/file
POST /api/v1/qr/number
POST /api/v1/qr/email
POST /api/v1/qr/sms
POST /api/v1/qr/geolocation
GET  /api/v1/
```

All generation endpoints accept `size` (128–2048 pixels) and `recovery` (`low`, `medium`, `high`, or `highest`). The defaults are `256` and `medium`. They also accept `show_logo` and `logo_png_base64`; alternatively, callers may use `multipart/form-data` and upload the PNG as `logo` or `logo_file`.

When `show_logo=true` and no custom PNG is supplied, the server uses `FALLBACK_LOGO_PATH`. Invalid custom PNGs return `400 Bad Request`; logo rendering never silently falls back to an unrelated image.

Successful responses contain the generated PNG inside the JSON response:

```json
{
  "qr_code": {
    "type": "url",
    "format": "png",
    "size": 256,
    "recovery": "medium",
    "logo_included": true,
    "logo_source": "fallback",
    "png_base64": "iVBORw0KGgo...",
    "data_uri": "data:image/png;base64,iVBORw0KGgo..."
  }
}
```

For complete request and response documentation, see [API documentation](docs/api.md) or retrieve the machine-readable [OpenAPI specification](docs/openapi.yaml) through `/openapi.yaml`. Release-candidate notes are available in [docs/releases/v1.0.0-rc.md](docs/releases/v1.0.0-rc.md).

## Configuration

| Variable | Required | Default | Description |
|---|---:|---|---|
| `API_KEY` | Yes | none | API key; at least 16 characters |
| `LISTEN_ADDRESS` | No | `:8080` | HTTP listen address |
| `READ_TIMEOUT` | No | `10s` | HTTP read timeout |
| `WRITE_TIMEOUT` | No | `10s` | HTTP write timeout |
| `IDLE_TIMEOUT` | No | `60s` | HTTP keep-alive timeout |
| `SHUTDOWN_TIMEOUT` | No | `15s` | Graceful shutdown timeout |
| `FALLBACK_LOGO_PATH` | No | `assets/fallback-logo.png` | PNG used when `show_logo=true` has no custom logo |
| `LOGO_MAX_BYTES` | No | `1048576` | Maximum custom or fallback PNG size |
| `OPENAPI_PATH` | No | `docs/openapi.yaml` | OpenAPI specification file served at `/openapi.yaml` |

Copy `.env.example` when preparing local environment configuration. `.env` is not loaded automatically; export the variables or use your process manager/container platform.

## Kubernetes

Create a Secret first rather than deploying the checked-in placeholder value:

```bash
kubectl -n default create secret generic qr-code-api \
  --from-literal=api-key="replace-with-a-cryptographically-random-key"

kubectl apply -f deploy/kubernetes/configmap.yaml
kubectl apply -f deploy/kubernetes/deployment.yaml
kubectl apply -f deploy/kubernetes/service.yaml
```

Or build and deploy the image, then use Kustomize:

```bash
docker build -t ghcr.io/faolanbig/qr-code-api-server:local .
kubectl apply -k deploy/kubernetes
```

Before production use, replace `secret.example.yaml` or omit it from `kustomization.yaml`, configure image registry credentials if needed, and use an ingress or gateway with TLS. See [deployment documentation](docs/deployment.md).

## Development

```bash
go mod download
go fmt ./...
go test ./...
go vet ./...
go build ./...
```

Tests cover payload formatting, PNG generation, authentication, validation, and HTTP responses.

## Project structure

```text
cmd/server/                 Application entry point
internal/config/            Environment configuration
internal/httpserver/        HTTP routing, middleware, handlers
internal/qrcode/            Payload builders and PNG rendering
assets/                     Default and fallback image assets
deploy/kubernetes/          Kubernetes manifests and Kustomize entry point
docs/                       API and deployment documentation
Dockerfile                  Multi-stage, non-root container image
LICENSE                     MIT license
```

## Security

- Never commit real API keys or production Secrets.
- Use a secret manager or Kubernetes Secret in production.
- Terminate TLS at an ingress, gateway, or load balancer.
- Restrict network access to trusted clients.
- Rotate keys if they are disclosed.
- The server intentionally has no persistent storage and does not log request bodies or API keys.

## Limitations

- File QR codes encode a data URI directly. They are suitable for small files only; the API rejects files larger than 1024 decoded bytes.
- QR payloads are limited by QR-code capacity. The selected library enforces its own maximum.
- No built-in horizontal scaling state is required; generation is stateless.
- Custom logos must be PNG images with dimensions between 16×16 and 1024×1024 pixels.
- The configured fallback logo is loaded at startup and must be a valid PNG.

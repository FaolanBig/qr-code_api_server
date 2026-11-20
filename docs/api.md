# API Reference

Base URL: `/api/v1`

The machine-readable OpenAPI 3.1 specification is available publicly at `/openapi.yaml` and in [openapi.yaml](openapi.yaml).

## Authentication

Use one of these headers for every API endpoint except `/healthz`, `/readyz`, and `/openapi.yaml`:

```text
X-API-Key: your-secret-api-key
```

or:

```text
Authorization: Bearer your-secret-api-key
```

The API key must contain at least 16 characters. Authentication failures return `401 Unauthorized`:

```json
{
  "error": {
    "code": "Unauthorized",
    "message": "invalid or missing API key"
  }
}
```

## Common generation fields

| Field | Type | Default | Description |
|---|---|---|---|
| `size` | integer | `256` | PNG width and height; range 128–2048 |
| `recovery` | string | `medium` | `low`, `medium`, `high`, or `highest` |
| `show_logo` | boolean | `false` | Place a PNG logo at the QR code center |
| `logo_png_base64` | string | none | PNG encoded as Base64 for JSON requests |

Unknown JSON fields are rejected. JSON request bodies are limited to 12 MiB. Multipart requests can upload a PNG in the `logo` or `logo_file` field. Custom PNGs are limited to `LOGO_MAX_BYTES` (default 1 MiB) and must be 16–1024 pixels wide and tall.

If `show_logo=true`, a custom uploaded or Base64 PNG is used first. If neither is supplied, `FALLBACK_LOGO_PATH` is used. If fallback logo configuration is missing or invalid, the server exits during startup. Invalid supplied PNGs return `400 Bad Request` rather than generating a logo-free QR code. Logos are rendered in a small clear zone and automatically use the `highest` recovery level for reliable scanning.

## Response

A successful generation request returns `200 OK`:

```json
{
  "qr_code": {
    "type": "text",
    "format": "png",
    "size": 256,
    "recovery": "medium",
    "logo_included": true,
    "logo_source": "fallback",
    "png_base64": "...",
    "data_uri": "data:image/png;base64,..."
  }
}
```

The API never returns a QR-code download URL. Clients can decode `png_base64` or render `data_uri` directly.

## Public probes

### Health

`GET /healthz`

```json
{"status":"ok"}
```

### Readiness

`GET /readyz`

```json
{"status":"ready"}
```

## API index

`GET /api/v1/` requires authentication.

## Logo examples

JSON request:

```json
{
  "url": "https://example.com",
  "show_logo": true,
  "logo_png_base64": "iVBORw0KGgo...",
  "size": 256,
  "recovery": "high"
}
```

Multipart request:

```bash
curl --fail --request POST http://localhost:8080/api/v1/qr/url \
  --header 'X-API-Key: replace-with-at-least-16-random-characters' \
  --form 'url=https://example.com' \
  --form 'show_logo=true' \
  --form 'size=256' \
  --form 'recovery=high' \
  --form 'logo=@./logo.png;type=image/png'
```

To use the server fallback logo:

```json
{
  "url": "https://example.com",
  "show_logo": true
}
```

When custom and fallback logos are both absent, `show_logo=true` returns `400 Bad Request`.

## URL

`POST /api/v1/qr/url`

```json
{
  "url": "https://example.com/products/1",
  "size": 256,
  "recovery": "medium"
}
```

Only absolute `http` and `https` URLs are accepted.

## vCard contact

`POST /api/v1/qr/vcard`

```json
{
  "first_name": "Ada",
  "last_name": "Lovelace",
  "organization": "Analytical Engines",
  "title": "Mathematician",
  "phone": "+44 1234 567890",
  "email": "ada@example.com",
  "website": "https://example.com",
  "address": "London",
  "size": 256,
  "recovery": "high"
}
```

At least `first_name` or `last_name` is required.

## Wi-Fi

`POST /api/v1/qr/wifi`

```json
{
  "ssid": "Office-Guest",
  "password": "a-strong-password",
  "security": "WPA",
  "hidden": false,
  "size": 256,
  "recovery": "high"
}
```

Allowed security values: `WPA`, `WEP`, `NOPASS`. Password is required for WPA/WEP. For an open network, omit `password` and use `NOPASS`.

## Calendar event

`POST /api/v1/qr/calendar`

Timed event:

```json
{
  "title": "Project review",
  "description": "Review the release",
  "location": "Conference Room A",
  "start": "2026-12-01T09:30:00Z",
  "end": "2026-12-01T10:30:00Z",
  "timezone": "Europe/London",
  "all_day": false,
  "size": 256,
  "recovery": "high"
}
```

Timed values use RFC 3339. All-day event:

```json
{
  "title": "Company holiday",
  "start": "2026-12-24",
  "end": "2026-12-25",
  "all_day": true
}
```

All-day values use `YYYY-MM-DD`. `end` is exclusive in some calendar applications for an all-day event.

## Plain text

`POST /api/v1/qr/text`

```json
{
  "text": "This text will be encoded directly.",
  "size": 256,
  "recovery": "medium"
}
```

## Small file

`POST /api/v1/qr/file`

```json
{
  "filename": "hello.txt",
  "mime_type": "text/plain",
  "base64_content": "aGVsbG8=",
  "size": 512,
  "recovery": "high"
}
```

The service decodes and re-encodes the supplied value as a data URI. Decoded files are limited to 1024 bytes and Base64 input is limited to 4096 characters. `mime_type` defaults to `application/octet-stream`.

## Number

`POST /api/v1/qr/number`

```json
{
  "number": "+1 (555) 123-4567",
  "size": 256,
  "recovery": "medium"
}
```

## Email

`POST /api/v1/qr/email`

```json
{
  "email": "hello@example.com",
  "subject": "Hello",
  "body": "Please send me more information.",
  "size": 256
}
```

## SMS

`POST /api/v1/qr/sms`

```json
{
  "phone": "+15551234567",
  "body": "Hello",
  "size": 256
}
```

## Geolocation

`POST /api/v1/qr/geolocation`

```json
{
  "latitude": 51.5074,
  "longitude": -0.1278,
  "label": "London",
  "size": 256
}
```

Latitude must be -90 to 90. Longitude must be -180 to 180.

## Error responses

Validation and bad-request errors use `400 Bad Request`:

```json
{
  "error": {
    "code": "Bad Request",
    "message": "url is required"
  }
}
```

Malformed JSON, unknown fields, trailing JSON values, and oversized bodies also use `400 Bad Request`. A request body larger than the configured limit may be reported as `Bad Request`.

# WhatsApp Notifier

Go service for sending notifications through WhatsApp. The project is based on
[`martinezsaweczko/go-api-template`](https://github.com/martinezsaweczko/go-api-template)
and includes an HTTP API, Swagger documentation, logging, Prometheus metrics,
and OpenTelemetry support.

## Status

The API foundation is in place. WhatsApp provider integration and notification
endpoints are the next implementation steps.

## Requirements

- Go 1.27.0
- `govulncheck` is invoked automatically via `go run` during `make build`

## Metrics

`make run` exposes Prometheus metrics at `http://localhost:8080/metrics`.

Key metrics:

| Metric | Labels | Description |
|--------|--------|-------------|
| `http_requests_total` | `method`, `path`, `status` | Total HTTP requests |
| `http_request_duration_seconds` | `method`, `path` | HTTP request latency |
| `whatsapp_connected` | — | WhatsApp client connected to server (0/1) |
| `whatsapp_logged_in` | — | WhatsApp client logged in (0/1) |
| `whatsapp_messages_sent_total` | `result` | Messages sent by result |
| `whatsapp_send_duration_seconds` | `result` | WhatsApp send latency |

The `result` label can be `success`, `disconnected`, `invalid_recipient`, or `provider_error`.

## Development

```bash
make build
make run
```

The API listens on `localhost:8080` by default. Swagger UI is available at
`http://localhost:8080/swagger/`.

On the first start, scan the QR code printed in the terminal from WhatsApp's
Linked devices screen. The linked session is stored in `whatsapp.db`; override
the location with `-whatsapp-session-db`.

Send a text notification:

```bash
curl -X POST http://localhost:8080/api/v1/notifications \
  -H 'Content-Type: application/json' \
  -d '{"recipient":"15551234567","message":"Hello from the notifier"}'
```

Run tests directly with:

```bash
go test ./...
```

## Next Steps

- Add delivery status handling, persistence, and retries.
- Add session management endpoints for headless deployments.

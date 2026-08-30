# Agent Notes — notifierWhatsapp

## Project facts

- Go 1.27.0, module `whatsapp-notifier`.
- Entry point: `cmd/main.go`.
- Configuration is flag-based, not env-based. See `config/config.go` for available flags.

## Build & dev

- **Build**: `make build`
  - Runs `swagger`, `check` (vet + fmt + govulncheck + mod tidy), then builds `build/main`.
  - Requires CGO because of `mattn/go-sqlite3`.
- **Run locally**: `make run` (builds then runs on `localhost:8080`).
- **Container**: `make container-build && make container-run` (uses `podman` by default; override with `CONTAINER_ENGINE=docker`).
- **Tests**: `go test -v -race -count=1 ./...` (matches CI).
- **Swagger regen**: `make swagger` → runs `swag init -g cmd/main.go -o docs`. Generated `docs/docs.go` is committed.

## WhatsApp integration

- Uses `go.mau.fi/whatsmeow` with a SQLite session store.
- Session DB path is set via `-whatsapp-session-db`.
- First start prints a QR code in the logs; scan it from WhatsApp **Linked devices**.
- `README.md` and `.github/.copilot-instructions` are stale and claim WhatsApp is not integrated — the code in `services/whatsapp.go` already integrates it.

## Release

- Push a semantic-version tag (`v*.*.*`) to trigger:
  - `.github/workflows/release.yml` — binaries.
  - `.github/workflows/container.yml` — `ghcr.io/martinezsaweczko/notifierwhatsapp:<version>` and `latest`.

## Container notes

- Runtime image is `gcr.io/distroless/base-debian12`.
- Runs as UID/GID `65532`.
- Exposes port `8080`.
- Needs a writable volume mounted at `/data` for the SQLite session DB.

## Style & process

- See `Claude.md` for project coding rules (error wrapping, contexts, testing, concurrency, etc.).
- CI enforces `go vet`, `gofmt`, `govulncheck`, tidy `go.mod`, and `-race` tests.

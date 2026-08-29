# syntax=docker/dockerfile:1

# Build stage
FROM golang:1.27.0-bookworm AS builder

WORKDIR /src

# Install C toolchain required for CGO (sqlite3)
RUN apt-get update && apt-get install -y --no-install-recommends \
    gcc \
    libc6-dev \
    && rm -rf /var/lib/apt/lists/*

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG VERSION=dev
ARG BUILD_TIME
ARG APP_NAME=whatsapp-notifier

ENV CGO_ENABLED=1
ENV GOOS=linux
ENV GOARCH=amd64

RUN go build \
    -ldflags "-X main.version=${VERSION} -X main.buildTime=${BUILD_TIME} -X main.appName=${APP_NAME}" \
    -o /bin/notifierwhatsapp \
    cmd/main.go

# Runtime stage
FROM gcr.io/distroless/base-debian12

WORKDIR /app

# Copy the binary from the builder stage
COPY --from=builder /bin/notifierwhatsapp /app/notifierwhatsapp

# Run as non-root user (distroless provides user 65532:65532)
USER 65532:65532

EXPOSE 8080

ENTRYPOINT ["/app/notifierwhatsapp"]
CMD ["-o11-prometheus-path=/metrics"]

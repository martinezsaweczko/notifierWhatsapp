
# Makefile for building and managing the Go application
# Set LDFlags for versioning and build information
GIT_VERSION := $(shell git describe --tags --always 2>/dev/null)
VERSION := $(if $(GIT_VERSION),$(GIT_VERSION),dev)
BUILD_TIME := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
APP_NAME := whatsapp-notifier
LDFLAGS := -X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME) -X main.appName=$(APP_NAME)

# Tooling is invoked via `go run` so contributors don't need to install binaries.
SWAG := go run github.com/swaggo/swag/cmd/swag@v1.16.6
VULNCHECK := go run golang.org/x/vuln/cmd/govulncheck@latest

# Container engine can be overridden to use docker or podman.
CONTAINER_ENGINE := podman
IMAGE_NAME := notifierwhatsapp
IMAGE_TAG := latest


.PHONY: build check clean keys clean-keys container-build container-run

keys: clean-keys
	@mkdir -p keys
	@echo "Generating RSA private key..."
	openssl genrsa -out keys/private.pem 2048
	@echo "Generating RSA public key..."
	openssl rsa -in keys/private.pem -pubout -out keys/public.pem
	@echo "Keys generated successfully in keys/ directory"

clean-keys:
	rm -rf keys

build: clean swagger check
	go build -ldflags "$(LDFLAGS)" -o build/main cmd/main.go

check:
	go vet ./...
	go fmt ./...
	$(VULNCHECK) ./...
	go mod tidy


clean: clean-swagger
	rm -rf build

clean-swagger:
	rm -rf docs

run: build
	./build/main -o11-prometheus-path=/metrics

vulncheck:
	$(VULNCHECK) ./...

swagger:
	mkdir -p docs && $(SWAG) init -g cmd/main.go -o docs

container-build: build
	$(CONTAINER_ENGINE) build \
		--build-arg VERSION=$(VERSION) \
		--build-arg BUILD_TIME=$(BUILD_TIME) \
		--build-arg APP_NAME=$(APP_NAME) \
		-t $(IMAGE_NAME):$(IMAGE_TAG) .

container-run:
	$(CONTAINER_ENGINE) run --rm \
		-p 8080:8080 \
		-v notifierwhatsapp-session:/data \
		$(IMAGE_NAME):$(IMAGE_TAG) \
		-http-address=0.0.0.0 \
		-whatsapp-session-db=/data/whatsapp.db \
		-o11-prometheus-path=/metrics

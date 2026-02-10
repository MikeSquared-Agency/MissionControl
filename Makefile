# MissionControl Makefile
VERSION ?= 0.5.0
PLATFORMS := darwin-amd64 darwin-arm64 linux-amd64 linux-arm64
DIST_DIR := dist

.PHONY: all build build-mc build-mc-noweb build-mc-core build-ci build-orchestrator build-web clean release test test-go test-rust test-web test-integration test-e2e test-all lint fmt

all: build

# Build all components
build: build-mc build-mc-core build-orchestrator build-web

# Build mc CLI (Go) - copies web dist for embedding
build-mc: build-web
	@echo "Building mc CLI..."
	@echo "Copying web UI for embedding..."
	rm -rf cmd/mc/dist
	cp -r web/dist cmd/mc/dist
	cd cmd/mc && go build -ldflags "-s -w -X main.version=$(VERSION)" -o ../../$(DIST_DIR)/mc .

# Build mc CLI without web UI embedding (for CI — no Node.js required)
build-mc-noweb: $(DIST_DIR)
	@echo "Building mc CLI (no web UI)..."
	mkdir -p cmd/mc/dist
	echo '<!doctype html><html><body>CI build - no UI</body></html>' > cmd/mc/dist/index.html
	cd cmd/mc && go build -ldflags "-s -w -X main.version=$(VERSION)" -o ../../$(DIST_DIR)/mc .

# Build mc for CI validation (no Node.js, no web UI)
# mc-core (Rust) is built separately; include it if cargo is available
build-ci: build-mc-noweb
	@if command -v cargo >/dev/null 2>&1; then $(MAKE) build-mc-core; else echo "cargo not found, skipping mc-core (use pre-built binary)"; fi
	@echo "CI build complete: $(DIST_DIR)/mc"

# Build mc-core (Rust)
build-mc-core:
	@echo "Building mc-core..."
	cd core && cargo build --release -p mc-core
	cp core/target/release/mc-core $(DIST_DIR)/mc-core

# Build orchestrator (Go)
build-orchestrator:
	@echo "Building orchestrator..."
	cd orchestrator && go build -ldflags "-s -w" -o ../$(DIST_DIR)/mc-orchestrator .

# Build web UI
build-web:
	@echo "Building web UI..."
	cd web && npm install && npm run build

# Run all tests
test: test-go test-rust test-web

test-go:
	@echo "Running Go tests..."
	cd cmd/mc && go test -v ./...
	cd orchestrator && go test -v ./...

test-rust:
	@echo "Running Rust tests..."
	cd core && cargo test

test-web:
	@echo "Running web tests..."
	cd web && npm test

# Integration tests (Go with build tag)
test-integration:
	@echo "Running Go integration tests..."
	cd orchestrator && go test -v -tags=integration ./...

# E2E tests (Playwright)
test-e2e:
	@echo "Running E2E tests..."
	cd web && npm run test:e2e

# Run all tests including integration and E2E
test-all: test lint test-integration test-e2e

# Lint all code
lint:
	@echo "Linting Go..."
	cd orchestrator && golangci-lint run || true
	cd cmd/mc && golangci-lint run || true
	@echo "Linting Rust..."
	cd core && cargo clippy -- -D warnings || true
	@echo "Linting TypeScript..."
	cd web && npm run lint || true

# Format all code
fmt:
	@echo "Formatting Go..."
	cd orchestrator && go fmt ./...
	cd cmd/mc && go fmt ./...
	@echo "Formatting Rust..."
	cd core && cargo fmt
	@echo "Formatting TypeScript..."
	cd web && npx prettier --write "src/**/*.{ts,tsx}" || true

# Clean build artifacts
clean:
	rm -rf $(DIST_DIR)
	rm -rf core/target
	rm -rf web/dist
	rm -rf web/node_modules

# Create distribution directory
$(DIST_DIR):
	mkdir -p $(DIST_DIR)

# Build release for a specific platform
release-darwin-amd64: $(DIST_DIR)
	@echo "Building for darwin/amd64..."
	GOOS=darwin GOARCH=amd64 go build -ldflags "-s -w -X main.version=$(VERSION)" -o $(DIST_DIR)/darwin-amd64/mc ./cmd/mc
	GOOS=darwin GOARCH=amd64 go build -ldflags "-s -w" -o $(DIST_DIR)/darwin-amd64/mc-orchestrator ./orchestrator
	cd core && cargo build --release -p mc-core --target x86_64-apple-darwin || echo "Cross-compile requires target: rustup target add x86_64-apple-darwin"
	cp core/target/x86_64-apple-darwin/release/mc-core $(DIST_DIR)/darwin-amd64/ 2>/dev/null || cp core/target/release/mc-core $(DIST_DIR)/darwin-amd64/

release-darwin-arm64: $(DIST_DIR)
	@echo "Building for darwin/arm64..."
	GOOS=darwin GOARCH=arm64 go build -ldflags "-s -w -X main.version=$(VERSION)" -o $(DIST_DIR)/darwin-arm64/mc ./cmd/mc
	GOOS=darwin GOARCH=arm64 go build -ldflags "-s -w" -o $(DIST_DIR)/darwin-arm64/mc-orchestrator ./orchestrator
	cd core && cargo build --release -p mc-core --target aarch64-apple-darwin || echo "Cross-compile requires target: rustup target add aarch64-apple-darwin"
	cp core/target/aarch64-apple-darwin/release/mc-core $(DIST_DIR)/darwin-arm64/ 2>/dev/null || cp core/target/release/mc-core $(DIST_DIR)/darwin-arm64/

release-linux-amd64: $(DIST_DIR)
	@echo "Building for linux/amd64..."
	GOOS=linux GOARCH=amd64 go build -ldflags "-s -w -X main.version=$(VERSION)" -o $(DIST_DIR)/linux-amd64/mc ./cmd/mc
	GOOS=linux GOARCH=amd64 go build -ldflags "-s -w" -o $(DIST_DIR)/linux-amd64/mc-orchestrator ./orchestrator
	cd core && cargo build --release -p mc-core --target x86_64-unknown-linux-gnu || echo "Cross-compile requires target: rustup target add x86_64-unknown-linux-gnu"

# Package releases as tarballs
package: $(DIST_DIR)
	@for platform in $(PLATFORMS); do \
		if [ -d "$(DIST_DIR)/$$platform" ]; then \
			echo "Packaging $$platform..."; \
			tar -czvf $(DIST_DIR)/mission-control-$(VERSION)-$$platform.tar.gz -C $(DIST_DIR)/$$platform .; \
		fi \
	done

# Full release build
release: clean $(DIST_DIR) release-darwin-arm64 release-darwin-amd64 package
	@echo "Release artifacts created in $(DIST_DIR)/"
	@ls -la $(DIST_DIR)/*.tar.gz 2>/dev/null || echo "No packages created yet"

# Install locally (macOS)
install: build
	@echo "Installing to /usr/local/bin..."
	cp $(DIST_DIR)/mc /usr/local/bin/
	cp $(DIST_DIR)/mc-core /usr/local/bin/
	cp $(DIST_DIR)/mc-orchestrator /usr/local/bin/
	@echo "Installed successfully!"

# Development: start both vite and orchestrator
dev:
	@echo "Starting development mode (vite + orchestrator)..."
	@echo "Press Ctrl+C to stop both services"
	@trap 'kill 0' INT; \
		(cd web && npm run dev) & \
		(cd orchestrator && go run . --workdir $(PWD)) & \
		wait

# Development: UI only (vite)
dev-ui:
	@echo "Starting Vite dev server..."
	cd web && npm run dev

# Development: API only (orchestrator)
dev-api:
	@echo "Starting orchestrator..."
	cd orchestrator && go run . --workdir $(PWD)

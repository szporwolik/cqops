.PHONY: build test lint clean run version install uninstall

VERSION := $(shell cat VERSION 2>/dev/null || echo "dev")
LDFLAGS := -s -w -X github.com/szporwolik/cqops/internal/version.Version=$(VERSION)
BUILD_DIR := build
BIN := $(BUILD_DIR)/cqops

# Default target
all: build

# Build for the current platform
build:
	@mkdir -p $(BUILD_DIR)
	go build -ldflags "$(LDFLAGS)" -o $(BIN) ./cmd/cqops/
	@echo "Built $(BIN) $(VERSION)"

# Build for all target platforms (cross-compile)
build-all:
	@mkdir -p $(BUILD_DIR)
	@echo "Building $(VERSION) for windows/amd64..."
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/cqops-windows-amd64.exe ./cmd/cqops/
	@echo "Building $(VERSION) for windows/arm64..."
	GOOS=windows GOARCH=arm64 CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/cqops-windows-arm64.exe ./cmd/cqops/
	@echo "Building $(VERSION) for linux/amd64..."
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/cqops-linux-amd64 ./cmd/cqops/
	@echo "Building $(VERSION) for linux/arm64..."
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/cqops-linux-arm64 ./cmd/cqops/
	@echo "Building $(VERSION) for darwin/amd64..."
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/cqops-darwin-amd64 ./cmd/cqops/
	@echo "Building $(VERSION) for darwin/arm64..."
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/cqops-darwin-arm64 ./cmd/cqops/
	@echo "Done. Binaries in $(BUILD_DIR)/"

# Install to system (creates menu entry + PATH)
install: build-all
	@case $$(uname -s) in \
		Linux)  scripts/install.sh ;; \
		Darwin) scripts/install.sh ;; \
		MINGW*|MSYS*|CYGWIN*) powershell -File scripts/install.ps1 ;; \
		*)      echo "Unsupported OS. Use 'make install-cli' for GOPATH install." ;; \
	esac

# Uninstall from system
uninstall:
	@case $$(uname -s) in \
		Linux|Darwin) scripts/uninstall.sh ;; \
		MINGW*|MSYS*|CYGWIN*) powershell -File scripts/uninstall.ps1 ;; \
		*) echo "Use scripts/uninstall.ps1 or scripts/uninstall.sh directly" ;; \
	esac

# CLI-only install to GOPATH/bin (no menu entry)
install-cli: build
	go install -ldflags "$(LDFLAGS)" ./cmd/cqops/
	@echo "Installed cqops $(VERSION) to GOPATH/bin"

# Run the app (builds first)
run: build
	./$(BIN)

# Run tests
test:
	go test -race -count=1 ./...

# Run tests with coverage
cover:
	go test -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Lint (requires golangci-lint installed)
lint:
	golangci-lint run ./...

# Format code
fmt:
	go fmt ./...

# Vet code
vet:
	go vet ./...

# Clean build artifacts
clean:
	rm -rf $(BUILD_DIR)
	go clean -cache -testcache

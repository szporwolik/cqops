.PHONY: build test lint clean run version install uninstall installer packages installer-all

VERSION := $(shell cat VERSION 2>/dev/null || echo "dev")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "")
BUILD_DATE := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -s -w -X github.com/szporwolik/cqops/internal/version.Version=$(VERSION) -X github.com/szporwolik/cqops/internal/version.Commit=$(COMMIT) -X github.com/szporwolik/cqops/internal/version.BuildDate=$(BUILD_DATE)
BUILD_DIR := build
BIN := $(BUILD_DIR)/cqops

all: build

build:
	@mkdir -p $(BUILD_DIR)
	@# Regenerate Windows .syso resources if go-winres is available
	@if command -v go-winres >/dev/null 2>&1; then \
		cd winres && go-winres make --product-version $(VERSION) --file-version $(VERSION) --in winres.json --out ../cmd/cqops/rsrc && cd ..; \
	fi
	go build -ldflags "$(LDFLAGS)" -o $(BIN) ./cmd/cqops/
	@echo "Built $(BIN) $(VERSION)"

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
	@echo "Building $(VERSION) for linux/armv7 (Pi Zero/1/2)..."
	GOOS=linux GOARCH=arm GOARM=7 CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/cqops-linux-armhf ./cmd/cqops/
	@echo "Building $(VERSION) for darwin/amd64..."
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/cqops-darwin-amd64 ./cmd/cqops/
	@echo "Building $(VERSION) for darwin/arm64..."
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/cqops-darwin-arm64 ./cmd/cqops/
	@echo "Done. Binaries in $(BUILD_DIR)/"

ifeq ($(OS),Windows_NT)
INSTALL = @powershell -File scripts/install.ps1
UNINSTALL = @powershell -File scripts/uninstall.ps1
else
INSTALL = @bash scripts/install.sh
UNINSTALL = @bash scripts/uninstall.sh
endif

install: build
	$(INSTALL)

uninstall:
	$(UNINSTALL)

install-cli: build
	go install -ldflags "$(LDFLAGS)" ./cmd/cqops/
	@echo "Installed cqops $(VERSION) to GOPATH/bin"

run: build
	./$(BIN)

test:
	go test -race -count=1 ./...

cover:
	go test -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

lint:
	golangci-lint run ./...

fmt:
	go fmt ./...

vet:
	go vet ./...

clean:
	rm -rf $(BUILD_DIR) dist
	go clean -cache -testcache

# ---------------------------------------------------------------------------
# Installers & packages
# ---------------------------------------------------------------------------

# Windows: NSIS installer (requires makensis on PATH)
installer:
ifeq ($(OS),Windows_NT)
	@powershell -File scripts/build-installer.ps1
else
	@echo "NSIS installer requires Windows. Use 'make packages' for Linux packages."
endif

# Linux: deb + rpm via nfpm (requires nfpm on PATH)
packages:
	@bash scripts/build-packages.sh

# All platforms: NSIS installer + Linux packages
installer-all:
ifeq ($(OS),Windows_NT)
	@powershell -File scripts/build-installer.ps1
endif
	@bash scripts/build-packages.sh

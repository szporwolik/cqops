.PHONY: build test lint clean run version install uninstall

VERSION := dev
LDFLAGS := -s -w -X github.com/szporwolik/cqops/internal/version.Version=
BUILD_DIR := build
BIN := /cqops

all: build

build:
	@mkdir -p 
	go build -ldflags "" -o  ./cmd/cqops/
	@echo "Built  "

build-all:
	@mkdir -p 
	@echo "Building  for windows/amd64..."
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "" -o /cqops-windows-amd64.exe ./cmd/cqops/
	@echo "Building  for windows/arm64..."
	GOOS=windows GOARCH=arm64 CGO_ENABLED=0 go build -ldflags "" -o /cqops-windows-arm64.exe ./cmd/cqops/
	@echo "Building  for linux/amd64..."
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "" -o /cqops-linux-amd64 ./cmd/cqops/
	@echo "Building  for linux/arm64..."
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -ldflags "" -o /cqops-linux-arm64 ./cmd/cqops/
	@echo "Building  for darwin/amd64..."
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "" -o /cqops-darwin-amd64 ./cmd/cqops/
	@echo "Building  for darwin/arm64..."
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -ldflags "" -o /cqops-darwin-arm64 ./cmd/cqops/
	@echo "Done. Binaries in /"

# Detect OS for install/uninstall
ifeq ($(OS),Windows_NT)
  INSTALL_CMD = @powershell -File scripts/install.ps1
  UNINSTALL_CMD = @powershell -File scripts/uninstall.ps1
else
  INSTALL_CMD = @bash scripts/install.sh
  UNINSTALL_CMD = @bash scripts/uninstall.sh
endif

install: build-all
	$(INSTALL_CMD)

uninstall:
	$(UNINSTALL_CMD)

install-cli: build
	go install -ldflags "$(LDFLAGS)" ./cmd/cqops/
	@echo "Installed cqops $(VERSION) to GOPATH/bin"

run: build
	./

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
	rm -rf 
	go clean -cache -testcache
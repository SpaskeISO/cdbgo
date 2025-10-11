.PHONY: all build test clean windows linux darwin freebsd help install bench

# Variables
OUTPUT_DIR ?= ./bin
LDFLAGS := # -s -w # <- Remove this line to enable binary stripping

# Colors for output
GREEN := \033[0;32m
BLUE := \033[0;34m
YELLOW := \033[0;33m
NC := \033[0m # No Color

# Default target
all: clean linux windows darwin

## help: Display this help message
help:
	@echo -e "CDB/CDB64 Build System"
	@echo -e ""
	@echo -e "Usage:"
	@echo -e "  make [target]"
	@echo -e ""
	@echo -e "Targets:"
	@awk '/^##/ {sub(/^## /, ""); desc=$$0; getline; printf "  $(BLUE)%-15s$(NC) %s\n", $$1, desc}' $(MAKEFILE_LIST)

## build: Build for current platform
build:
	@echo -e "$(BLUE)Building for current platform...$(NC)"
	@mkdir -p $(OUTPUT_DIR)
	go build -ldflags="$(LDFLAGS)" -o $(OUTPUT_DIR)/cdb ./cmd/cdb/
	go build -ldflags="$(LDFLAGS)" -o $(OUTPUT_DIR)/cdb64 ./cmd/cdb64/
	@echo -e "$(GREEN)✓ Build complete$(NC)"

## windows: Build for Windows (amd64, 386, arm64)
windows:
	@echo -e "$(BLUE)Building for Windows...$(NC)"
	@mkdir -p $(OUTPUT_DIR)
	GOOS=windows GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o $(OUTPUT_DIR)/cdb-windows-amd64.exe ./cmd/cdb/
	GOOS=windows GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o $(OUTPUT_DIR)/cdb64-windows-amd64.exe ./cmd/cdb64/
	GOOS=windows GOARCH=386 go build -ldflags="$(LDFLAGS)" -o $(OUTPUT_DIR)/cdb-windows-386.exe ./cmd/cdb/
	GOOS=windows GOARCH=386 go build -ldflags="$(LDFLAGS)" -o $(OUTPUT_DIR)/cdb64-windows-386.exe ./cmd/cdb64/
	GOOS=windows GOARCH=arm64 go build -ldflags="$(LDFLAGS)" -o $(OUTPUT_DIR)/cdb-windows-arm64.exe ./cmd/cdb/
	GOOS=windows GOARCH=arm64 go build -ldflags="$(LDFLAGS)" -o $(OUTPUT_DIR)/cdb64-windows-arm64.exe ./cmd/cdb64/
	@echo -e "$(GREEN)✓ Windows builds complete$(NC)"

## linux: Build for Linux (amd64, arm64, arm)
linux:
	@echo -e "$(BLUE)Building for Linux...$(NC)"
	@mkdir -p $(OUTPUT_DIR)
	GOOS=linux GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o $(OUTPUT_DIR)/cdb-linux-amd64 ./cmd/cdb/
	GOOS=linux GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o $(OUTPUT_DIR)/cdb64-linux-amd64 ./cmd/cdb64/
	GOOS=linux GOARCH=arm64 go build -ldflags="$(LDFLAGS)" -o $(OUTPUT_DIR)/cdb-linux-arm64 ./cmd/cdb/
	GOOS=linux GOARCH=arm64 go build -ldflags="$(LDFLAGS)" -o $(OUTPUT_DIR)/cdb64-linux-arm64 ./cmd/cdb64/
	GOOS=linux GOARCH=arm go build -ldflags="$(LDFLAGS)" -o $(OUTPUT_DIR)/cdb-linux-arm ./cmd/cdb/
	GOOS=linux GOARCH=arm go build -ldflags="$(LDFLAGS)" -o $(OUTPUT_DIR)/cdb64-linux-arm ./cmd/cdb64/
	@echo -e "$(GREEN)✓ Linux builds complete$(NC)"

## darwin: Build for macOS (Intel and Apple Silicon)
darwin:
	@echo -e "$(BLUE)Building for macOS...$(NC)"
	@mkdir -p $(OUTPUT_DIR)
	GOOS=darwin GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o $(OUTPUT_DIR)/cdb-darwin-amd64 ./cmd/cdb/
	GOOS=darwin GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o $(OUTPUT_DIR)/cdb64-darwin-amd64 ./cmd/cdb64/
	GOOS=darwin GOARCH=arm64 go build -ldflags="$(LDFLAGS)" -o $(OUTPUT_DIR)/cdb-darwin-arm64 ./cmd/cdb/
	GOOS=darwin GOARCH=arm64 go build -ldflags="$(LDFLAGS)" -o $(OUTPUT_DIR)/cdb64-darwin-arm64 ./cmd/cdb64/
	@echo -e "$(GREEN)✓ macOS builds complete$(NC)"

## freebsd: Build for FreeBSD (64-bit)
freebsd:
	@echo -e "$(BLUE)Building for FreeBSD...$(NC)"
	@mkdir -p $(OUTPUT_DIR)
	GOOS=freebsd GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o $(OUTPUT_DIR)/cdb-freebsd-amd64 ./cmd/cdb/
	GOOS=freebsd GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o $(OUTPUT_DIR)/cdb64-freebsd-amd64 ./cmd/cdb64/
	@echo -e "$(GREEN)✓ FreeBSD builds complete$(NC)"

## static: Build statically linked binaries (no CGO dependencies)
static:
	@echo -e "$(BLUE)Building static binaries (no CGO)...$(NC)"
	@mkdir -p $(OUTPUT_DIR)
	@echo -e "  Linux (fully static)..."
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o $(OUTPUT_DIR)/cdb-linux-amd64-static ./cmd/cdb/
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o $(OUTPUT_DIR)/cdb64-linux-amd64-static ./cmd/cdb64/
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags="$(LDFLAGS)" -o $(OUTPUT_DIR)/cdb-linux-arm64-static ./cmd/cdb/
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags="$(LDFLAGS)" -o $(OUTPUT_DIR)/cdb64-linux-arm64-static ./cmd/cdb64/
	CGO_ENABLED=0 GOOS=linux GOARCH=arm go build -ldflags="$(LDFLAGS)" -o $(OUTPUT_DIR)/cdb-linux-arm-static ./cmd/cdb/
	CGO_ENABLED=0 GOOS=linux GOARCH=arm go build -ldflags="$(LDFLAGS)" -o $(OUTPUT_DIR)/cdb64-linux-arm-static ./cmd/cdb64/
	@echo -e "  Windows (no CGO)..."
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o $(OUTPUT_DIR)/cdb-windows-amd64-static.exe ./cmd/cdb/
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o $(OUTPUT_DIR)/cdb64-windows-amd64-static.exe ./cmd/cdb64/
	CGO_ENABLED=0 GOOS=windows GOARCH=386 go build -ldflags="$(LDFLAGS)" -o $(OUTPUT_DIR)/cdb-windows-386-static.exe ./cmd/cdb/
	CGO_ENABLED=0 GOOS=windows GOARCH=386 go build -ldflags="$(LDFLAGS)" -o $(OUTPUT_DIR)/cdb64-windows-386-static.exe ./cmd/cdb64/
	CGO_ENABLED=0 GOOS=windows GOARCH=arm64 go build -ldflags="$(LDFLAGS)" -o $(OUTPUT_DIR)/cdb-windows-arm64-static.exe ./cmd/cdb/
	CGO_ENABLED=0 GOOS=windows GOARCH=arm64 go build -ldflags="$(LDFLAGS)" -o $(OUTPUT_DIR)/cdb64-windows-arm64-static.exe ./cmd/cdb64/
	@echo -e "$(YELLOW)  Note: macOS does not support fully static binaries$(NC)"
	@echo -e "$(GREEN)✓ Static builds complete$(NC)"

## test: Run all tests
test:
	@echo -e "$(BLUE)Running tests...$(NC)"
	go test -v ./...
	@echo -e "$(GREEN)✓ Tests complete$(NC)"

## bench: Run benchmarks
bench:
	@echo -e "$(BLUE)Running benchmarks...$(NC)"
	@echo -e ""
	@echo -e "$(YELLOW)CDB32 Benchmarks:$(NC)"
	go test -bench=. -benchtime=1s ./cdb/ -run=^$$
	@echo -e ""
	@echo -e "$(YELLOW)CDB64 Benchmarks:$(NC)"
	go test -bench=. -benchtime=1s ./cdb/cdb64/ -run=^$$
	@echo -e ""
	@echo -e "$(GREEN)✓ Benchmarks complete$(NC)"

## collision: Run collision analysis
collision:
	@echo -e "$(BLUE)Running collision analysis...$(NC)"
	@echo -e ""
	@echo -e "$(YELLOW)CDB32 Collision Analysis:$(NC)"
	go test -v -run=TestCollisionAnalysis ./cdb/
	@echo -e ""
	@echo -e "$(YELLOW)CDB64 Collision Analysis:$(NC)"
	go test -v -run=TestCollisionAnalysis ./cdb/cdb64/
	@echo -e ""
	@echo -e "$(GREEN)✓ Collision analysis complete$(NC)"

## install: Install binaries to GOPATH/bin
install:
	@echo -e "$(BLUE)Installing binaries...$(NC)"
	go install ./cmd/cdb/
	go install ./cmd/cdb64/
	@echo -e "$(GREEN)✓ Installation complete$(NC)"

## clean: Remove build artifacts
clean:
	@echo -e "$(BLUE)Cleaning build artifacts...$(NC)"
	rm -rf $(OUTPUT_DIR)
	go clean
	@echo -e "$(GREEN)✓ Clean complete$(NC)"

## dist: Create distribution archives (regular and static builds)
dist: all static
	@echo -e "$(BLUE)Creating distribution archives...$(NC)"
	@mkdir -p dist
	@cp LICENSE $(OUTPUT_DIR)/
	# Windows (regular)
	cd $(OUTPUT_DIR) && zip ../dist/cdb-windows-amd64.zip LICENSE cdb-windows-amd64.exe cdb64-windows-amd64.exe && cd ..
	cd $(OUTPUT_DIR) && zip ../dist/cdb-windows-386.zip LICENSE cdb-windows-386.exe cdb64-windows-386.exe && cd ..
	cd $(OUTPUT_DIR) && zip ../dist/cdb-windows-arm64.zip LICENSE cdb-windows-arm64.exe cdb64-windows-arm64.exe && cd ..
	# Windows (static)
	cd $(OUTPUT_DIR) && zip ../dist/cdb-windows-amd64-static.zip LICENSE cdb-windows-amd64-static.exe cdb64-windows-amd64-static.exe && cd ..
	cd $(OUTPUT_DIR) && zip ../dist/cdb-windows-386-static.zip LICENSE cdb-windows-386-static.exe cdb64-windows-386-static.exe && cd ..
	cd $(OUTPUT_DIR) && zip ../dist/cdb-windows-arm64-static.zip LICENSE cdb-windows-arm64-static.exe cdb64-windows-arm64-static.exe && cd ..
	# Linux (regular)
	cd $(OUTPUT_DIR) && tar -czf ../dist/cdb-linux-amd64.tar.gz LICENSE cdb-linux-amd64 cdb64-linux-amd64 && cd ..
	cd $(OUTPUT_DIR) && tar -czf ../dist/cdb-linux-arm64.tar.gz LICENSE cdb-linux-arm64 cdb64-linux-arm64 && cd ..
	cd $(OUTPUT_DIR) && tar -czf ../dist/cdb-linux-arm.tar.gz LICENSE cdb-linux-arm cdb64-linux-arm && cd ..
	# Linux (static)
	cd $(OUTPUT_DIR) && tar -czf ../dist/cdb-linux-amd64-static.tar.gz LICENSE cdb-linux-amd64-static cdb64-linux-amd64-static && cd ..
	cd $(OUTPUT_DIR) && tar -czf ../dist/cdb-linux-arm64-static.tar.gz LICENSE cdb-linux-arm64-static cdb64-linux-arm64-static && cd ..
	cd $(OUTPUT_DIR) && tar -czf ../dist/cdb-linux-arm-static.tar.gz LICENSE cdb-linux-arm-static cdb64-linux-arm-static && cd ..
	# macOS
	cd $(OUTPUT_DIR) && tar -czf ../dist/cdb-darwin-amd64.tar.gz LICENSE cdb-darwin-amd64 cdb64-darwin-amd64 && cd ..
	cd $(OUTPUT_DIR) && tar -czf ../dist/cdb-darwin-arm64.tar.gz LICENSE cdb-darwin-arm64 cdb64-darwin-arm64 && cd ..
	@echo -e "$(GREEN)✓ Distribution archives created in dist/$(NC)"
	@echo -e ""
	@echo -e "Distribution packages:"
	@ls -lh dist/

## list: List all compiled binaries
list:
	@echo -e "Compiled binaries in $(OUTPUT_DIR):"
	@ls -lh $(OUTPUT_DIR)/ 2>/dev/null || echo -e "No binaries found. Run 'make all' first."

## size: Show binary sizes
size:
	@echo -e "Binary sizes:"
	@ls -lh $(OUTPUT_DIR)/ 2>/dev/null | grep -E "(cdb|cdb64)" | awk '{printf "  %-35s %s\n", $$9, $$5}' || echo -e "No binaries found."

## fmt: Format Go code
fmt:
	@echo -e "$(BLUE)Formatting code...$(NC)"
	go fmt ./...
	@echo -e "$(GREEN)✓ Format complete$(NC)"

## vet: Run go vet
vet:
	@echo -e "$(BLUE)Running go vet...$(NC)"
	go vet ./...
	@echo -e "$(GREEN)✓ Vet complete$(NC)"

## mod: Tidy and verify modules
mod:
	@echo -e "$(BLUE)Tidying modules...$(NC)"
	go mod tidy
	go mod verify
	@echo -e "$(GREEN)✓ Module operations complete$(NC)"

## check: Run all checks (fmt, vet, test)
check: fmt vet test
	@echo -e "$(GREEN)✓ All checks passed$(NC)"


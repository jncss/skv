.PHONY: build install clean test help windows windows-32 windows-64 all-platforms

# Build the skv CLI binary for current platform
build:
	@echo "Building skv CLI..."
	go build -o skv .
	@echo "✓ Build complete: ./skv"

# Build for Windows 64-bit
windows-64:
	@echo "Building for Windows 64-bit..."
	GOOS=windows GOARCH=amd64 go build -o skv-windows-amd64.exe .
	@echo "✓ Build complete: ./skv-windows-amd64.exe"

# Build for Windows 32-bit
windows-32:
	@echo "Building for Windows 32-bit..."
	GOOS=windows GOARCH=386 go build -o skv-windows-386.exe .
	@echo "✓ Build complete: ./skv-windows-386.exe"

# Build both Windows versions
windows: windows-32 windows-64
	@echo "✓ All Windows builds complete"

# Build for all platforms
all-platforms: build windows
	@echo "Building for Linux 64-bit..."
	GOOS=linux GOARCH=amd64 go build -o skv-linux-amd64 .
	@echo "✓ Build complete: ./skv-linux-amd64"
	@echo "Building for macOS 64-bit..."
	GOOS=darwin GOARCH=amd64 go build -o skv-darwin-amd64 .
	@echo "✓ Build complete: ./skv-darwin-amd64"
	@echo "Building for macOS ARM64..."
	GOOS=darwin GOARCH=arm64 go build -o skv-darwin-arm64 .
	@echo "✓ Build complete: ./skv-darwin-arm64"
	@echo "✓ All platform builds complete"

# Install skv to GOPATH/bin
install:
	@echo "Installing skv CLI..."
	go install .
	@echo "✓ Installed to $(shell go env GOPATH)/bin/skv"

# Clean build artifacts
clean:
	@echo "Cleaning..."
	rm -f skv skv-*.exe skv-linux-* skv-darwin-*
	@echo "✓ Clean complete"

# Run tests
test:
	@echo "Running tests..."
	go test -v
	@echo "✓ Tests complete"

# Show help
help:
	@echo "SKV CLI - Build Commands"
	@echo ""
	@echo "Usage:"
	@echo "  make build        - Build the skv binary for current platform"
	@echo "  make windows-32   - Build Windows 32-bit executable"
	@echo "  make windows-64   - Build Windows 64-bit executable"
	@echo "  make windows      - Build both Windows executables"
	@echo "  make all-platforms- Build for all platforms (Linux, macOS, Windows)"
	@echo "  make install      - Install skv to GOPATH/bin"
	@echo "  make clean        - Remove build artifacts"
	@echo "  make test         - Run tests"
	@echo "  make help         - Show this help message"
	@echo ""
	@echo "Cross-compilation outputs:"
	@echo "  skv-windows-386.exe    - Windows 32-bit"
	@echo "  skv-windows-amd64.exe  - Windows 64-bit"
	@echo "  skv-linux-amd64        - Linux 64-bit"
	@echo "  skv-darwin-amd64       - macOS Intel"
	@echo "  skv-darwin-arm64       - macOS Apple Silicon"
	@echo ""
	@echo "After building:"
	@echo "  ./skv help        - Show CLI usage"
	@echo ""

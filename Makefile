.PHONY: clean build install test test-verbose test-coverage test-all help

# Default target
help:
	@echo "Available targets:"
	@echo "  clean        - Remove all built artifacts"
	@echo "  build        - Build nclip and nclipd binaries"
	@echo "  install      - Install binaries and start/restart daemon service"
	@echo "  test         - Run all unit tests"
	@echo "  test-verbose - Run all unit tests with verbose output"
	@echo "  test-coverage - Run all unit tests with coverage report"
	@echo "  test-all     - Run tests, check formatting, and tidy modules"
	@echo "  help         - Show this help message"

# Clean all built artifacts
clean:
	@echo "Cleaning built artifacts..."
	rm -f nclip nclipd
	@echo "Clean complete."

# Build both binaries
build:
	@echo "Building nclip (TUI)..."
	go build -o nclip ./cmd
	@echo "Building nclipd (daemon)..."
	go build -o nclipd ./cmd/nclipd
	@echo "Build complete."

# Install binaries and manage daemon service
install: build
	@echo "Stopping service..."
	systemctl --user stop nclip

	@echo "Installing binaries..."
	mkdir -p ~/.local/bin
	cp nclip ~/.local/bin/nclip
	cp nclipd ~/.local/bin/nclipd


	@echo "Installing systemd service..."
	mkdir -p ~/.config/systemd/user
	cp templates/systemd/nclip.service ~/.config/systemd/user/
	
	@echo "Reloading systemd and managing service..."
	systemctl --user daemon-reload
	
	@if systemctl --user is-active --quiet nclip; then \
		echo "Restarting existing nclip service..."; \
		systemctl --user restart nclip; \
	else \
		echo "Enabling and starting nclip service..."; \
		systemctl --user enable nclip; \
		systemctl --user start nclip; \
	fi
	
	@echo "Installation complete."
	@echo "Service status:"
	@systemctl --user status nclip --no-pager -l

# Run all unit tests
test:
	@echo "Running unit tests..."
	go test ./...

# Run all unit tests with verbose output
test-verbose:
	@echo "Running unit tests (verbose)..."
	go test -v ./...

# Run all unit tests with coverage report
test-coverage:
	@echo "Running unit tests with coverage..."
	go test -cover ./...
	@echo ""
	@echo "For detailed coverage by module:"
	@echo "  internal/config:    Configuration management and TOML parsing"
	@echo "  internal/storage:   SQLite database operations and clipboard items"
	@echo "  internal/security:  Threat detection and security hash management"
	@echo "  internal/clipboard: Clipboard monitoring and copy operations"
	@echo "  internal/ui:        UI utility functions and core logic"

# Run comprehensive testing with code quality checks
test-all:
	@echo "Running comprehensive test suite..."
	@echo "1. Running unit tests..."
	go test ./...
	@echo ""
	@echo "2. Checking code formatting..."
	@if [ -n "$$(gofmt -l .)" ]; then \
		echo "Code formatting issues found:"; \
		gofmt -l .; \
		echo "Run 'go fmt ./...' to fix formatting"; \
		exit 1; \
	else \
		echo "Code formatting: OK"; \
	fi
	@echo ""
	@echo "3. Running static analysis..."
	go vet ./...
	@echo ""
	@echo "4. Tidying go modules..."
	go mod tidy
	@echo ""
	@echo "5. Running tests with coverage..."
	go test -cover ./...
	@echo ""
	@echo "All checks passed! ✅"

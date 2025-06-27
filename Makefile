.PHONY: clean build install help

# Default target
help:
	@echo "Available targets:"
	@echo "  clean   - Remove all built artifacts"
	@echo "  build   - Build nclip and nclipdaemond binaries"
	@echo "  install - Install binaries and start/restart daemon service"
	@echo "  help    - Show this help message"

# Clean all built artifacts
clean:
	@echo "Cleaning built artifacts..."
	rm -f nclip nclipdaemon
	@echo "Clean complete."

# Build both binaries
build:
	@echo "Building nclip (TUI)..."
	go build -o nclip ./cmd
	@echo "Building nclipdaemon (daemon)..."
	go build -o nclipdaemon ./cmd/daemon
	@echo "Build complete."

# Install binaries and manage daemon service
install: build
	@echo "Stopping service..."
	systemctl --user stop nclip

	@echo "Installing binaries..."
	mkdir -p ~/.local/bin
	cp nclip ~/.local/bin/nclip
	cp nclipdaemon ~/.local/bin/nclipdaemon


	@echo "Installing systemd service..."
	mkdir -p ~/.config/systemd/user
	cp nclip.service ~/.config/systemd/user/
	
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

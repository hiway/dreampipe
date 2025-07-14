\
# Makefile for dreampipe

# Variables
BINARY_NAME=dreampipe
CMD_PATH=./cmd/dreampipe
INSTALL_DIR=$(HOME)/bin

# Get version from git tags, fallback to dev
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS=-ldflags "-s -w -X main.version=$(VERSION)"

# Targets

## setup: Install required development tools (Go)
.PHONY: setup
setup:
	@echo "Setting up development environment..."
	@OS=$$(uname -s); \
	case $$OS in \
		Linux) \
			if command -v apt-get >/dev/null 2>&1; then \
				echo "Detected Ubuntu/Debian Linux"; \
				echo "Installing dependencies..."; \
				sudo apt-get update; \
				sudo apt-get install -y curl wget git build-essential; \
				if ! command -v go >/dev/null 2>&1; then \
					if [ -d "/usr/local/go" ]; then \
						echo "Error: Found existing Go installation at /usr/local/go but 'go' command is not available."; \
						echo "This suggests a broken or incomplete installation."; \
						echo "Please manually remove /usr/local/go and reinstall Go from: https://go.dev/doc/install"; \
						exit 1; \
					fi; \
					echo "Installing Go following official instructions from https://go.dev/doc/install"; \
					echo "Downloading and installing Go..."; \
					curl -fsSL https://golang.org/dl/go1.21.5.linux-amd64.tar.gz | sudo tar -C /usr/local -xzf -; \
					echo ""; \
					echo "Add /usr/local/go/bin to your PATH by adding this line to your ~/.profile or ~/.bashrc:"; \
					echo "export PATH=\$$PATH:/usr/local/go/bin"; \
					echo ""; \
					echo "Then run: source ~/.profile  (or restart your terminal)"; \
				else \
					echo "Go is already installed: $$(go version)"; \
					if [ -d "/usr/local/go" ]; then \
						echo "Note: To upgrade Go, please follow the official instructions at: https://go.dev/doc/install"; \
					fi; \
				fi; \
			else \
				echo "Unsupported Linux distribution. Please install Go manually from: https://go.dev/"; \
				exit 1; \
			fi; \
			;; \
		Darwin) \
			echo "Detected macOS"; \
			if ! command -v brew >/dev/null 2>&1; then \
				echo "Error: Homebrew is not installed."; \
				echo "Please install Homebrew from: https://brew.sh/"; \
				echo "Then run 'make setup' again."; \
				exit 1; \
			fi; \
			echo "Installing dependencies..."; \
			brew install go git; \
			;; \
		FreeBSD) \
			echo "Detected FreeBSD"; \
			echo "Installing dependencies..."; \
			sudo pkg update; \
			sudo pkg install -y go git; \
			;; \
		*) \
			echo "Unsupported operating system: $$OS"; \
			echo "Please install Go manually from: https://go.dev/"; \
			exit 1; \
			;; \
	esac
	@echo "GoReleaser installation:"
	@if ! command -v goreleaser >/dev/null 2>&1; then \
		echo "GoReleaser is not installed. Please install it from: https://goreleaser.com/install/"; \
		echo "Note: GoReleaser is only required for release builds, not for regular development."; \
	else \
		echo "GoReleaser is already installed: $$(goreleaser --version)"; \
	fi
	@echo "Setup completed!"
	@echo ""
	@echo "Next steps:"
	@echo "1. If Go was just installed, restart your terminal or source your shell profile"
	@echo "2. Verify installation: make check-deps"
	@echo "3. Build the project: make build"
	@echo "4. For releases, install GoReleaser from: https://goreleaser.com/install/"

## check-deps: Check if required dependencies are installed
.PHONY: check-deps
check-deps:
	@echo "Checking development dependencies..."
	@if command -v go >/dev/null 2>&1; then \
		echo "✓ Go: $$(go version)"; \
	else \
		echo "✗ Go is not installed"; \
		exit 1; \
	fi
	@if command -v goreleaser >/dev/null 2>&1; then \
		echo "✓ GoReleaser: $$(goreleaser --version | head -1)"; \
	else \
		echo "⚠ GoReleaser is not installed (optional for development)"; \
		echo "  Install from: https://goreleaser.com/install/"; \
	fi
	@if command -v git >/dev/null 2>&1; then \
		echo "✓ Git: $$(git --version)"; \
	else \
		echo "✗ Git is not installed"; \
		exit 1; \
	fi
	@echo "All dependencies are installed!"

## clean: Remove the built binary and dist directory
.PHONY: clean
clean:
	@echo "Cleaning up..."
	@rm -f $(BINARY_NAME)
	@rm -rf dist/

## run: Run the application, passing through arguments
# Example: make run ARGS="your instruction here"
# Example: echo "hello" | make run ARGS="translate to pirate"
.PHONY: run
run:
	@echo "Running dreampipe..."
	@go run $(CMD_PATH) $(ARGS)

## build: Build the application for the current OS and architecture
.PHONY: build
build:
	@echo "Building $(BINARY_NAME) version $(VERSION)..."
	@go build $(LDFLAGS) -o $(BINARY_NAME) $(CMD_PATH)
	@echo "$(BINARY_NAME) built successfully."

## build-all: Build the application for all supported OS and architectures
.PHONY: build-all
build-all:
	@echo "Building $(BINARY_NAME) for all supported platforms..."
	@mkdir -p dist
	@GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)-linux-amd64 $(CMD_PATH)
	@GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)-linux-arm64 $(CMD_PATH)
	@GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)-darwin-amd64 $(CMD_PATH)
	@GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)-darwin-arm64 $(CMD_PATH)
	@GOOS=freebsd GOARCH=amd64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)-freebsd-amd64 $(CMD_PATH)
	@GOOS=freebsd GOARCH=arm64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)-freebsd-arm64 $(CMD_PATH)
	@echo "All binaries built successfully in dist/"

## test-release: Test the release process locally using GoReleaser
.PHONY: test-release
test-release:
	@echo "Testing release process locally..."
	@goreleaser release --snapshot --clean
	@echo "Test release completed. Check dist/ directory for artifacts."

## release: Create and push a new release tag (requires VERSION)
# Example: make release VERSION=v1.0.0
.PHONY: release
release:
	@if [ -z "$(VERSION)" ]; then \
		echo "Error: VERSION is required. Usage: make release VERSION=v1.0.0"; \
		exit 1; \
	fi
	@echo "Creating release $(VERSION)..."
	@if git tag -l | grep -q "^$(VERSION)$$"; then \
		echo "Error: Tag $(VERSION) already exists"; \
		exit 1; \
	fi
	@if [ -n "$$(git status --porcelain)" ]; then \
		echo "Error: Working directory is not clean. Please commit or stash changes."; \
		exit 1; \
	fi
	@echo "Checking if we're on main branch..."
	@if [ "$$(git branch --show-current)" != "main" ]; then \
		echo "Error: Must be on main branch to create a release"; \
		exit 1; \
	fi
	@echo "Pulling latest changes..."
	@git pull origin main
	@echo "Creating tag $(VERSION)..."
	@git tag -a $(VERSION) -m "Release $(VERSION)"
	@echo "Pushing tag to GitHub..."
	@git push origin $(VERSION)
	@echo "Release $(VERSION) created and pushed successfully!"
	@echo "Monitor the release at: https://github.com/hiway/dreampipe/actions"

## version: Show the current version
.PHONY: version
version:
	@echo "Version: $(VERSION)"

## install: Install the application system-wide to /usr/local/bin
.PHONY: install
install: build
	@echo "Installing $(BINARY_NAME) to /usr/local/bin..."
	@sudo mkdir -p /usr/local/bin
	@sudo cp $(BINARY_NAME) /usr/local/bin/$(BINARY_NAME)
	@echo "$(BINARY_NAME) installed to /usr/local/bin/$(BINARY_NAME)"

## installuser: Install the application to $(INSTALL_DIR)
.PHONY: installuser
installuser: build
	@echo "Installing $(BINARY_NAME) to $(INSTALL_DIR)..."
	@mkdir -p $(INSTALL_DIR)
	@cp $(BINARY_NAME) $(INSTALL_DIR)/$(BINARY_NAME)
	@echo "$(BINARY_NAME) installed to $(INSTALL_DIR)/$(BINARY_NAME)"

## uninstall: Remove the application from /usr/local/bin
.PHONY: uninstall
uninstall:
	@echo "Removing $(BINARY_NAME) from /usr/local/bin..."
	@sudo rm -f /usr/local/bin/$(BINARY_NAME)
	@echo "$(BINARY_NAME) removed from /usr/local/bin"

## uninstalluser: Remove the application from $(INSTALL_DIR)
.PHONY: uninstalluser
uninstalluser:
	@echo "Removing $(BINARY_NAME) from $(INSTALL_DIR)..."
	@rm -f $(INSTALL_DIR)/$(BINARY_NAME)
	@echo "$(BINARY_NAME) removed from $(INSTALL_DIR)"

## install-examples: Install example scripts to $(INSTALL_DIR)
.PHONY: install-examples
install-examples:
	@echo "Installing example scripts to $(INSTALL_DIR)..."
	@mkdir -p $(INSTALL_DIR)
	@for example_file in examples/*.md; do \
		filename=$$(basename "$$example_file" .md); \
		echo "Installing $$filename to $(INSTALL_DIR)/$$filename..."; \
		cp "$$example_file" "$(INSTALL_DIR)/$$filename"; \
		chmod +x "$(INSTALL_DIR)/$$filename"; \
	done
	@echo "Example scripts installed."

## uninstall-examples: Remove example scripts from $(INSTALL_DIR)
.PHONY: uninstall-examples
uninstall-examples:
	@echo "Removing example scripts from $(INSTALL_DIR)..."
	@mkdir -p $(INSTALL_DIR)
	@for example_file in examples/*.md; do \
		filename=$$(basename "$$example_file" .md); \
		installed_file="$(INSTALL_DIR)/$$filename"; \
		if [ -f "$$installed_file" ]; then \
			if cmp -s "$$example_file" "$$installed_file"; then \
				echo "Removing $$filename from $(INSTALL_DIR)..."; \
				rm -f "$$installed_file"; \
			else \
				if [ "$(FORCE)" = "true" ]; then \
					echo "Force removing modified $$filename from $(INSTALL_DIR)..."; \
					rm -f "$$installed_file"; \
				else \
					echo "Warning: $$filename has been modified, skipping removal. Use FORCE=true to remove anyway."; \
				fi; \
			fi; \
		else \
			echo "$$filename not found in $(INSTALL_DIR), skipping..."; \
		fi; \
	done
	@echo "Example scripts removal completed."

## help: Show this help message
.PHONY: help
help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@echo "  setup               Install required development tools (Go)"
	@echo "  check-deps          Check if required dependencies are installed"
	@echo "  clean               Remove the built binary and dist directory"
	@echo "  run                 Run the application, passing through arguments"
	@echo "  build               Build the application for the current OS and architecture"
	@echo "  build-all           Build the application for all supported OS and architectures"
	@echo "  install             Install the application system-wide to /usr/local/bin"
	@echo "  installuser         Install the application to $(INSTALL_DIR)"
	@echo "  uninstall           Remove the application from /usr/local/bin"
	@echo "  uninstalluser       Remove the application from $(INSTALL_DIR)"
	@echo "  install-examples    Install example scripts to $(INSTALL_DIR)"
	@echo "  uninstall-examples  Remove example scripts from $(INSTALL_DIR)"
	@echo "  test-release        Test the release process locally using GoReleaser"
	@echo "  release             Create and push a new release tag (requires VERSION)"
	@echo "  version             Show the current version"
	@echo "  help                Show this help message"
	@echo ""
	@echo "Options:"
	@echo "  FORCE=true          Force removal of modified examples in uninstall-examples"
	@echo "  VERSION=x.y.z       Override version for builds"

.DEFAULT_GOAL := help

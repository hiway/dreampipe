\
# Makefile for dreampipe

# Variables
BINARY_NAME=dreampipe
CMD_PATH=./cmd/dreampipe/main.go
INSTALL_DIR=$(HOME)/bin

# Targets

## clean: Remove the built binary
.PHONY: clean
clean:
	@echo "Cleaning up..."
	@rm -f $(BINARY_NAME)

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
	@echo "Building $(BINARY_NAME)..."
	@go build -o $(BINARY_NAME) $(CMD_PATH)
	@echo "$(BINARY_NAME) built successfully."

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

## help: Show this help message
.PHONY: help
help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@echo "  clean               Remove the built binary"
	@echo "  run                 Run the application, passing through arguments"
	@echo "  build               Build the application for the current OS and architecture"
	@echo "  install             Install the application system-wide to /usr/local/bin"
	@echo "  installuser         Install the application to $(INSTALL_DIR)"
	@echo "  uninstall           Remove the application from /usr/local/bin"
	@echo "  uninstalluser       Remove the application from $(INSTALL_DIR)"
	@echo "  install-examples    Install example scripts to $(INSTALL_DIR)"
	@echo "  help                Show this help message"

.DEFAULT_GOAL := help

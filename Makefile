# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOMOD=$(GOCMD) mod
BINARY_NAME=shadow-empire-bot
DOCKER_IMAGE=ghcr.io/1solon/shadow-empire-pbem-bot
DOCKER_TAG=latest
BUILD_DIR=build

# Main target
all: deps build

# Build the application
build: setup-build
	$(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME) -v

# Create build directory
setup-build:
	mkdir -p $(BUILD_DIR)

# Run the application
run:
	$(BUILD_DIR)/$(BINARY_NAME)

# Tidy up dependencies
deps:
	$(GOMOD) tidy

# Test the application
# TODO: Add more tests
test:
	$(GOTEST) -v ./...

# Clean build files
clean:
	$(GOCLEAN)
	rm -rf $(BUILD_DIR)

# Build Docker image
docker-build:
	docker build -t $(DOCKER_IMAGE):$(DOCKER_TAG) .

# Run Docker container with mounted data directory
docker-run:
	docker run -d \		-e USER_MAPPINGS="Player1 123456789012345678,Player2 234567890123456789" \
		-e GAME_NAME="PBEM1" \
		-e DISCORD_WEBHOOK_URL="https://discord.com/api/webhooks/your-webhook-url" \
		-e REMINDER_INTERVAL_MINUTES="720" \
		-v "./data:/app/data" \
		$(DOCKER_IMAGE):$(DOCKER_TAG)

# Stop all running containers of this image
docker-stop:
	docker stop $$(docker ps -q --filter ancestor=$(DOCKER_IMAGE):$(DOCKER_TAG)) 2>/dev/null || true

# Create necessary directories
setup: setup-build
	mkdir -p data

# Help command
help:
	@echo "Make targets:"
	@echo "  all          - Build the application with dependencies"
	@echo "  build        - Build the application to $(BUILD_DIR) directory"
	@echo "  run          - Run the application"
	@echo "  deps         - Tidy up dependencies"
	@echo "  test         - Run tests"
	@echo "  clean        - Clean build files"
	@echo "  docker-build - Build Docker image"
	@echo "  docker-run   - Run Docker container"
	@echo "  docker-stop  - Stop Docker containers"
	@echo "  setup        - Create necessary directories"
	@echo "  help         - Show this help"

.PHONY: all build setup-build run deps test clean docker-build docker-run docker-stop setup help

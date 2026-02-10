# Makefile for exchange-backend
# Single source of truth for protobuf files

# Proto source directories
PROTO_DIR := proto
PROTO_GEN_DIR := proto/gen

# Proto files (add new services here)
PROTO_FILES := auth.proto
# PROTO_FILES += orders.proto  # Uncomment when orders.proto is added

# Go module path
GO_MODULE := exchange-backedn

# K8s output
K8S_KONG_DIR := k8s/infrastructure/kong
K8S_PROTO_CONFIGMAP := $(K8S_KONG_DIR)/proto-configmap.yaml

.PHONY: all proto proto-gen proto-clean k8s-proto-configmap clean help

all: proto k8s-proto-configmap

help:
	@echo "Available targets:"
	@echo "  proto                  - Generate Go code from proto files"
	@echo "  proto-clean            - Remove generated proto files"
	@echo "  k8s-proto-configmap    - Generate K8s ConfigMap from proto files"
	@echo "  clean                  - Clean all generated files"
	@echo "  docker-up              - Start all services with docker-compose"
	@echo "  docker-down            - Stop all services"
	@echo "  docker-logs            - Show logs from all services"

# =============================================================================
# Proto Generation
# =============================================================================

proto: proto-clean proto-gen

proto-gen:
	@echo "Generating Go code from proto files..."
	@mkdir -p $(PROTO_GEN_DIR)/auth
	protoc \
		--proto_path=$(PROTO_DIR) \
		--go_out=$(PROTO_GEN_DIR)/auth --go_opt=paths=source_relative \
		--go-grpc_out=$(PROTO_GEN_DIR)/auth --go-grpc_opt=paths=source_relative \
		--grpc-gateway_out=$(PROTO_GEN_DIR)/auth --grpc-gateway_opt=paths=source_relative \
		$(PROTO_DIR)/auth.proto
	@echo "Proto generation complete: $(PROTO_GEN_DIR)"

proto-clean:
	@echo "Cleaning generated proto files..."
	@rm -rf $(PROTO_GEN_DIR)

# =============================================================================
# K8s ConfigMap Generation (single source of truth for Kong protos)
# =============================================================================

k8s-proto-configmap:
	@echo "Generating K8s proto ConfigMap from source protos..."
	@./scripts/generate-proto-configmap.sh

# =============================================================================
# Docker Compose
# =============================================================================

docker-up:
	@echo "Starting services..."
	docker compose -f deploy/docker/docker-compose.yml up -d

docker-down:
	@echo "Stopping services..."
	docker compose -f deploy/docker/docker-compose.yml down

docker-logs:
	docker compose -f deploy/docker/docker-compose.yml logs -f

docker-build:
	@echo "Building services..."
	docker compose -f deploy/docker/docker-compose.yml build

docker-restart: docker-down docker-up

# =============================================================================
# Development
# =============================================================================

dev-setup: proto k8s-proto-configmap
	@echo "Development setup complete!"

# =============================================================================
# Clean
# =============================================================================

clean: proto-clean
	@echo "Cleaning all generated files..."
	@rm -f $(K8S_PROTO_CONFIGMAP)

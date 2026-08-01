# file-upload-service

This repository contains a Go-based file upload service.

## Structure

- `api/` - OpenAPI/Swagger definitions and generated API docs
- `cmd/api/` - API server entrypoint
- `configs/` - Configuration management
- `deployments/docker/` - Docker deployment manifests
- `deployments/k8s/` - Kubernetes manifests
- `docs/` - Architecture, ERD, and API documentation
- `internal/` - Application implementation details
- `pkg/` - Shared libraries
- `scripts/` - Utility scripts and helpers
- `test/` - End-to-end and integration tests

## Getting Started

1. Copy `.env.example` to `.env`
2. Customize `configs/config.yaml`
3. Run `go run ./cmd/api`

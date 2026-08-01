# File Upload Service Roadmap (Go + MinIO)

## Goal

Build a production-style file upload service using Go, MinIO,
PostgreSQL, Kafka and Docker.

## Architecture

-   API: Go (Gin/Fiber)
-   Object Storage: MinIO
-   Metadata DB: PostgreSQL
-   Queue: Kafka
-   Cache: Redis (later)
-   Workers:
    -   Virus Scan
    -   OCR
    -   Thumbnail
    -   Preview
    -   Notification

## Sprint 0 - Project Setup

### Deliverables

-   Initialize Go project
-   Docker Compose
-   PostgreSQL
-   MinIO
-   Kafka
-   Basic project structure
-   Health Check API

## Sprint 1 - Metadata Service

### Features

-   PostgreSQL connection
-   Migration
-   File metadata CRUD
-   Repository pattern

## Sprint 2 - Upload API

### APIs

-   POST /files/upload
-   POST /files/presigned-upload \### Tasks
-   Upload to MinIO
-   Save metadata
-   Validation

## Sprint 3 - Download API

### APIs

-   GET /files/{id}
-   GET /files/presigned-download

## Sprint 4 - Presigned URL

### Tasks

-   Generate Upload URL
-   Generate Download URL
-   Expiration

## Sprint 5 - Kafka Integration

### Tasks

-   Producer
-   Consumer
-   Publish upload events

## Sprint 6 - Virus Scan Worker

-   Consume upload event
-   Scan file
-   Update metadata

## Sprint 7 - Thumbnail Worker

-   Generate thumbnail
-   Upload thumbnail to MinIO
-   Update metadata

## Sprint 8 - OCR Worker

-   Extract text
-   Save OCR result

## Sprint 9 - Preview Worker

-   PDF preview
-   Image preview
-   Video preview

## Sprint 10 - Notification Worker

-   Email
-   WebSocket
-   Push Notification

## Sprint 11 - Security

-   JWT
-   RBAC
-   Rate Limit
-   Audit Log

## Sprint 12 - Monitoring

-   Prometheus
-   Grafana
-   Structured Logging
-   OpenTelemetry

## Sprint 13 - Performance

-   Benchmark
-   Load Test
-   Multipart Upload
-   File Deduplication

## Sprint 14 - Kubernetes

-   Deployment
-   Service
-   Ingress
-   HPA

## Definition of Done

-   Unit Tests
-   Swagger/OpenAPI
-   Docker Compose
-   README
-   Logging
-   Error Handling
-   CI/CD Ready

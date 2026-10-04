# File Upload Service — Architecture (Open-Source Stack)

## Goal

Build a **production-style File Upload Service** using Go and 100% open-source infrastructure —
replacing AWS S3 → **MinIO**, AWS RDS → **PostgreSQL**, AWS SQS/SNS → **Apache Kafka**, CDN → **Nginx**, Lambda → **Go Workers**.

---

## System Architecture Diagram

```
                  ┌──────────────────────────────────────────────────────────────┐
                  │                    ASYNC WORKERS (Consumers)                 │
                  │  ┌─────────────┐  ┌──────────────────┐  ┌────────────────┐  │
                  │  │ Virus Scan  │  │ Thumbnail Worker  │  │ Preview Worker │  │
                  │  │ (ClamAV)    │  │  (imaging/ffmpeg) │  │  (LibreOffice) │  │
                  │  └──────┬──────┘  └────────┬─────────┘  └───────┬────────┘  │
                  │         │                  │                     │           │
                  │  ┌──────┴──────┐  ┌────────┴─────────┐  ┌───────┴────────┐  │
                  │  │ OCR Worker  │  │ Notification Wkr  │  │  Validation    │  │
                  │  │ (Tesseract) │  │  (SMTP/WebSocket) │  │  Worker        │  │
                  │  └─────────────┘  └──────────────────┘  └────────────────┘  │
                  └──────────────────────────────┬───────────────────────────────┘
                                                 │ consume events
                                                 ▼
┌─────────┐    /files/presigned-upload   ┌──────────────────────┐    enqueue     ┌──────────────────┐
│         │ ─────────────────────────►   │                      │ ─────────────► │                  │
│         │ ◄─────────────────────────   │                      │                │  Apache Kafka    │
│         │   return presigned PUT URL   │   File Upload API    │                │  (KRaft mode)    │
│         │                             │     (Go net/http)     │                │                  │
│ Clients │    /files/upload ────────►   │                      │                └──────────────────┘
│         │ ◄──────────────────────────  │                      │
│         │                             │   /files/callback ◄──┼─── (client calls after direct upload)
│         │    /files/{FILE_ID}──────►   │                      │
│         │ ◄────────────────────────── │                      │    save/query   ┌──────────────────┐
│         │                             │                      │ ◄────────────── │                  │
│         │    /files/presigned-download │                      │ ──────────────► │   PostgreSQL     │
└─────────┘ ─────────────────────────►  └───────────┬──────────┘                │  (Metadata DB)   │
                                                     │                           └──────────────────┘
                                         upload/download objects
                                                     │
                               ┌─────────────────────┼─────────────────────┐
                               │                     ▼                     │
                               │         ┌──────────────────────┐          │
                               │         │        MinIO         │          │
                               │         │  (Object Storage)    │          │
                               │         │   S3-compatible API  │          │
                               │         └──────────────────────┘          │
                               └─────────────────────────────────────────  ┘
                                              (direct upload)
                                                     ▲
                                    Presigned PUT URL │ Client uploads directly
                                                     │
                               ┌──────────────────────────────┐
                               │          Nginx               │
                               │   (CDN / Reverse Proxy /     │
                               │    Static File Cache)        │
                               │  good for public/static files│
                               └──────────────────────────────┘
                                             ▲
                                   /files/{FILE_ID}
                                             │
                                         Clients
```

---

## Functional Requirements

| Feature | Status |
|---|:---:|
| Upload / Download files | ✅ |
| File metadata (name, size, type, checksum) | ✅ |
| Large file support (100MB+) | ✅ |
| Multipart / Chunked upload | ⏳ Sprint 13 |
| Pause / Resume upload | ⏳ Sprint 13 |
| File deduplication (SHA-256) | ⏳ Sprint 13 |
| Retention policy | ⏳ Sprint 13 |
| Security (JWT, RBAC, Rate Limit) | ⏳ Sprint 11 |
| Async processing (Workers via Kafka) | ⏳ Sprint 5-10 |

---

## Tech Stack (Open-Source)

| Role | Cloud Original | Open-Source Replacement |
|---|---|---|
| **API Server** | AWS API Gateway + ECS | Go (`net/http`, std lib) |
| **Object Storage** | AWS S3 / Azure Blob | **MinIO** (S3-compatible) |
| **Metadata Database** | AWS RDS (PostgreSQL) | **PostgreSQL** (Docker) |
| **Message Queue / Event Broker** | AWS SQS / SNS / EventBridge | **Apache Kafka** (KRaft mode) |
| **CDN / Static Cache** | AWS CloudFront / Cloudflare | **Nginx** (Reverse Proxy + Cache) |
| **Virus Scanning** | AWS GuardDuty | **ClamAV** |
| **Thumbnail / Image Processing** | AWS Lambda + Sharp | Go **`imaging`** + **`ffmpeg`** |
| **OCR** | AWS Textract | **Tesseract OCR** |
| **Preview Generation** | AWS Lambda | **LibreOffice headless** + **ffmpeg** |
| **Email Notification** | AWS SES | **SMTP** (MailHog local / SMTP relay) |
| **WebSocket / Push Notification** | AWS API Gateway WebSocket | Go **gorilla/websocket** |
| **Container Orchestration** | AWS EKS | **Kubernetes (k3s / kind)** |
| **Monitoring & Metrics** | AWS CloudWatch | **Prometheus + Grafana** |
| **Distributed Tracing** | AWS X-Ray | **OpenTelemetry + Jaeger** |
| **Cache (future)** | AWS ElastiCache (Redis) | **Redis** |
| **CI/CD** | AWS CodePipeline | **GitHub Actions / GitLab CI** |

---

## API Design

### Upload Flows

#### Flow 1 — Direct Upload (small/medium files ≤ 100MB)
```
Client ──POST /api/v1/files/upload──► API ──► MinIO (stream)
                                          ──► PostgreSQL (metadata)
                                          ──► Kafka (file.uploaded event)
```

#### Flow 2 — Presigned Upload (large files, upload directly to MinIO)
```
Client ──POST /api/v1/files/presigned-upload──► API ──► MinIO (generate PUT URL)
                                                    ◄── presigned PUT URL + fileID

Client ──PUT {presignedURL}──► MinIO (direct, bypass API)

Client ──POST /api/v1/files/callback──► API ──► PostgreSQL (save metadata)
                                            ──► Kafka (file.uploaded event)
```

#### Flow 3 — Download / Presigned Download (static / large files)
```
Client ──GET /api/v1/files/{id}──► API ──► MinIO (stream download)
Client ──GET /api/v1/files/{id}/presigned-download──► API ──► MinIO (generate GET URL)
                                                          ◄── presigned GET URL
Client ──GET {presignedURL}──► MinIO (direct download)
```

### Route Table

| Method | Route | Description | Sprint |
|---|---|---|---|
| `POST` | `/api/v1/files/upload` | Direct upload | Sprint 2 ✅ |
| `GET` | `/api/v1/files/{id}` | Get metadata | Sprint 2 ✅ |
| `POST` | `/api/v1/files/presigned-upload` | Get presigned PUT URL | Sprint 4 ⏳ |
| `POST` | `/api/v1/files/callback` | Confirm presigned upload | Sprint 4 ⏳ |
| `GET` | `/api/v1/files/{id}/download` | Stream download | Sprint 3 ⏳ |
| `GET` | `/api/v1/files/{id}/presigned-download` | Get presigned GET URL | Sprint 4 ⏳ |
| `DELETE` | `/api/v1/files/{id}` | Delete file | Sprint 11 ⏳ |
| `GET` | `/healthz` | Health check (PG + MinIO + Kafka) | Sprint 0 ✅ |

---

## Kafka Event Design

| Topic | Producer | Consumer(s) | Payload |
|---|---|---|---|
| `file.uploaded` | File Upload API | Virus Scan, Thumbnail, OCR, Preview, Notification Workers | `{fileID, bucket, objectKey, mimeType, size}` |
| `file.scanned` | Virus Scan Worker | File Upload API (update status) | `{fileID, status: clean/infected}` |
| `file.thumbnail.done` | Thumbnail Worker | File Upload API (update metadata) | `{fileID, thumbnailKey}` |
| `file.ocr.done` | OCR Worker | File Upload API (save OCR text) | `{fileID, text}` |

---

## Data Model — `files` table

```sql
CREATE TABLE files (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name          TEXT NOT NULL,
    original_name TEXT NOT NULL,
    mime_type     TEXT NOT NULL,
    size          BIGINT NOT NULL,
    bucket        TEXT NOT NULL,
    object_key    TEXT NOT NULL UNIQUE,
    status        TEXT NOT NULL DEFAULT 'uploaded',
    -- status: uploaded | scanning | clean | infected | processing | ready | failed
    checksum      TEXT NOT NULL,        -- SHA-256
    uploaded_by   TEXT,
    thumbnail_key TEXT,                 -- set by Thumbnail Worker
    ocr_text      TEXT,                 -- set by OCR Worker
    preview_key   TEXT,                 -- set by Preview Worker
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

---

## Async Workers Architecture

```
                Apache Kafka
                Topic: file.uploaded
                      │
        ┌─────────────┼──────────────────────────┐
        │             │             │             │
        ▼             ▼             ▼             ▼
 ┌────────────┐ ┌──────────────┐ ┌──────────┐ ┌────────────────┐
 │ Virus Scan │ │  Thumbnail   │ │   OCR    │ │    Preview     │
 │  Worker    │ │   Worker     │ │  Worker  │ │    Worker      │
 │ (ClamAV)   │ │ (imaging/    │ │(Tesserat)│ │ (LibreOffice / │
 │            │ │  ffmpeg)     │ │          │ │  ffmpeg)       │
 └──────┬─────┘ └──────┬───────┘ └────┬─────┘ └───────┬────────┘
        │              │              │               │
        │   Download from MinIO, process, upload result to MinIO
        │              │              │               │
        └──────────────┴──────────────┴───────────────┘
                                │
                   Publish to file.*.done topic
                                │
                       File Upload API
                    (update DB metadata + status)
                                │
                  Notify client via WebSocket (future)
```

---

## Sprint Roadmap

| Sprint | Goal | Key Deliverables |
|---|---|---|
| **Sprint 0** | Project Setup | Go project, Docker Compose (PG + MinIO + Kafka), Health Check `/healthz` |
| **Sprint 1** | Metadata Service | PostgreSQL conn, Migration, CRUD Repository pattern |
| **Sprint 2** | Upload API | `POST /files/upload`, MIME detection, SHA-256 checksum, save metadata |
| **Sprint 3** | Download API | `GET /files/{id}`, `GET /files/{id}/download` stream |
| **Sprint 4** | Presigned URL | Presigned Upload URL, Presigned Download URL, `/callback` handler |
| **Sprint 5** | Kafka Integration | Producer (emit `file.uploaded`), Consumer framework |
| **Sprint 6** | Virus Scan Worker | ClamAV consumer, update `status` in DB |
| **Sprint 7** | Thumbnail Worker | `imaging`/`ffmpeg` consumer, upload thumbnail to MinIO |
| **Sprint 8** | OCR Worker | Tesseract consumer, save extracted text to DB |
| **Sprint 9** | Preview Worker | LibreOffice/ffmpeg consumer, generate preview, upload to MinIO |
| **Sprint 10** | Notification Worker | SMTP email + gorilla/websocket push |
| **Sprint 11** | Security | JWT auth, RBAC, Rate Limiting, Audit Log, `DELETE /files/{id}` |
| **Sprint 12** | Monitoring | Prometheus metrics, Grafana dashboard, OpenTelemetry + Jaeger tracing, structured logging (zerolog) |
| **Sprint 13** | Performance | Multipart/chunked upload, pause/resume, file deduplication, benchmarks, load tests |
| **Sprint 14** | Kubernetes | k3s/kind deployment, Service, Ingress, HPA, resource limits |

---

## Definition of Done (per Sprint)

- [ ] Unit Tests (≥ 80% coverage on business logic)
- [ ] Swagger / OpenAPI spec updated
- [ ] Docker Compose updated if new service added
- [ ] Structured logging (zerolog)
- [ ] Proper error handling (no naked `error` returns)
- [ ] README / docs updated
- [ ] CI passes (build + test + lint)

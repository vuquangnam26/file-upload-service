# Giải Thích Chi Tiết Kiến Trúc — File Upload Service

> Tài liệu này giải thích từng thành phần trong hệ thống, vai trò của nó, cách nó hoạt động,
> và tại sao cần phải có nó. Stack sử dụng 100% open-source, không phụ thuộc cloud provider.

---

## Sơ Đồ Tổng Quan

```
                  ┌──────────────────────────────────────────────────────────────┐
                  │                    ASYNC WORKERS (Consumers)                 │
                  │  ┌─────────────┐  ┌──────────────────┐  ┌────────────────┐  │
                  │  │ Virus Scan  │  │ Thumbnail Worker  │  │ Preview Worker │  │
                  │  │ (ClamAV)    │  │  (imaging/ffmpeg) │  │  (LibreOffice) │  │
                  │  └─────────────┘  └──────────────────┘  └────────────────┘  │
                  │  ┌─────────────┐  ┌──────────────────┐  ┌────────────────┐  │
                  │  │ OCR Worker  │  │ Notification Wkr  │  │  Validation    │  │
                  │  │ (Tesseract) │  │  (SMTP/WebSocket) │  │  Worker        │  │
                  │  └─────────────┘  └──────────────────┘  └────────────────┘  │
                  └──────────────────────────┬───────────────────────────────────┘
                                             │ consume events
                                             ▼
┌─────────┐   /files/presigned-upload  ┌──────────────────────┐  enqueue  ┌───────────────┐
│         │ ────────────────────────►  │                      │ ─────────►│               │
│         │ ◄────────────────────────  │                      │           │ Apache Kafka  │
│         │  presigned PUT URL         │   File Upload API    │           │ (KRaft mode)  │
│         │                            │     (Go net/http)    │           │               │
│ Clients │   /files/upload ─────────► │                      │           └───────────────┘
│         │ ◄─────────────────────────  │                      │
│         │                            │   /files/callback ◄──┼── client gọi sau khi upload
│         │   /files/{FILE_ID} ──────►  │                      │  save/query  ┌────────────────┐
│         │ ◄────────────────────────   │                      │ ◄──────────► │                │
│         │                            │                      │              │   PostgreSQL   │
│         │  /files/presigned-download  └──────────┬───────────┘              │ (Metadata DB)  │
└─────────┘ ─────────────────────────►             │                          └────────────────┘
                                          upload/download
                                                   │
                                                   ▼
                              ┌─────────────────────────────────┐
                              │             MinIO               │
                              │      (Object Storage)           │
                              │      S3-compatible API          │
                              └─────────────────────────────────┘
                                            ▲
                          Presigned PUT URL │ client upload trực tiếp
                                            │
                              ┌─────────────────────────────────┐
                              │             Nginx               │
                              │  (CDN / Reverse Proxy / Cache)  │
                              │    good for public/static files  │
                              └─────────────────────────────────┘
```

---

## 1. Clients (Người dùng cuối)

### Là gì?
Bất kỳ thứ gì gọi vào hệ thống: web browser, mobile app, CLI tool, service khác (microservice).

### Làm gì?
Clients thực hiện 3 hành động chính:

| Hành động | Route | Mô tả |
|---|---|---|
| Upload file trực tiếp | `POST /files/upload` | Gửi file thẳng lên API, API xử lý và đẩy lên MinIO |
| Upload file lớn (qua Presigned URL) | `POST /files/presigned-upload` → `PUT {url}` | Lấy URL đặc biệt, rồi upload thẳng vào MinIO, bypass API |
| Tải file xuống | `GET /files/{FILE_ID}` hoặc presigned download URL | Tải file hoặc lấy link download |

### Tại sao cần 2 luồng upload khác nhau?

> **Vấn đề:** Nếu mọi file đều đi qua API server, server phải nhận toàn bộ dữ liệu file vào RAM/disk rồi mới đẩy lên MinIO.
> Với file 1GB, điều đó có nghĩa API server phải giữ 1GB trong bộ nhớ — rất nguy hiểm với nhiều request đồng thời.

- **Direct Upload** phù hợp cho file nhỏ (< 100MB): đơn giản, 1 request duy nhất.
- **Presigned Upload** phù hợp cho file lớn: client upload thẳng vào MinIO, API server không bị nghẽn.

---

## 2. File Upload API (Go `net/http`)

### Là gì?
Đây là **trái tim** của hệ thống — một HTTP server viết bằng Go, dùng standard library `net/http`.

### Làm gì?
- Nhận request từ client
- Validate file (kích thước, MIME type qua magic bytes)
- Tính SHA-256 checksum để chống trùng lặp và kiểm tra toàn vẹn dữ liệu
- Giao tiếp với MinIO để lưu file
- Giao tiếp với PostgreSQL để lưu metadata
- Phát event lên Kafka sau khi upload thành công
- Tạo Presigned URL (Upload/Download) từ MinIO SDK
- Expose `/healthz` để kiểm tra toàn bộ hệ thống

### Tại sao dùng Go `net/http` thay vì framework (Gin, Fiber)?

| Tiêu chí | Go `net/http` | Gin / Fiber |
|---|---|---|
| Phụ thuộc | Zero dependencies | Thêm external deps |
| Performance | Cực tốt (routing Go 1.22+) | Tốt nhưng thêm overhead |
| Pattern matching | ✅ Go 1.22+ hỗ trợ `GET /files/{id}` | ✅ |
| Học được gì | Hiểu sâu HTTP internals | Dùng abstraction |

> **Kết luận:** Dùng `net/http` chuẩn để học nguyên lý, dễ migrate sang framework sau nếu cần.

### Tại sao phải detect MIME bằng magic bytes?

```
Client gửi: Content-Type: image/jpeg
Thực tế:    File là .exe (virus)
```

Client hoàn toàn có thể giả mạo `Content-Type` header. Magic bytes là 512 byte đầu tiên của file, mỗi định dạng có pattern riêng (ví dụ: JPEG luôn bắt đầu bằng `FF D8 FF`). Go có `http.DetectContentType()` đọc magic bytes → không thể giả mạo.

### Tại sao tính SHA-256 checksum?

- **Chống trùng lặp:** 2 file có cùng SHA-256 → cùng nội dung → không cần lưu 2 lần (deduplication).
- **Kiểm tra toàn vẹn:** Client download file rồi tính lại SHA-256 → so sánh với DB → biết file có bị corrupt không.
- **Audit:** Pháp lý/compliance yêu cầu biết file đã bị thay đổi chưa.

---

## 3. MinIO (Object Storage — thay thế AWS S3)

### Là gì?
MinIO là một **object storage server** mã nguồn mở, tương thích 100% API với AWS S3.
Nghĩa là bất kỳ code nào dùng AWS S3 SDK đều chạy được với MinIO mà không cần sửa code.

### Làm gì?
- Lưu trữ **file thực sự** (binary data): ảnh, video, PDF, v.v.
- Phân loại theo `bucket` (tương tự folder gốc) và `object key` (đường dẫn trong bucket).
- Tạo **Presigned URL**: URL tạm thời có chữ ký, hết hạn sau X giây, cho phép upload/download mà không cần xác thực API.
- Hỗ trợ multipart upload cho file lớn.
- Expose web console ở port `9001`.

### Tại sao không lưu file vào PostgreSQL hoặc local disk?

| Lưu ở đâu | Vấn đề |
|---|---|
| PostgreSQL (BYTEA) | DB không thiết kế để lưu binary lớn → chậm, tốn RAM, khó scale |
| Local disk (API server) | Server restart mất file; không scale ngang được (2 instance → 2 disk khác nhau) |
| **MinIO (Object Storage)** | ✅ Thiết kế đúng cho binary lớn, scale được, redundant, có API chuẩn |

### Object Key Pattern

File được lưu theo cấu trúc:
```
uploads/{year}/{month}/{day}/{uuid}_{sanitized_filename}{ext}
# Ví dụ:
uploads/2026/10/03/550e8400-e29b-41d4-a716-446655440000_invoice.pdf
```

> **Tại sao chia theo ngày?** MinIO/S3 performance giảm nếu 1 prefix có hàng triệu object. Chia theo ngày giúp phân tán đều.

### Presigned URL là gì và hoạt động thế nào?

```
1. Client: "Tôi muốn upload file X"
2. API: Gọi MinIO SDK → tạo URL dạng:
   https://minio:9000/bucket/uploads/2026/10/03/xxx.pdf
   ?X-Amz-Algorithm=AWS4-HMAC-SHA256
   &X-Amz-Credential=...
   &X-Amz-Date=...
   &X-Amz-Expires=3600          ← hết hạn sau 1 giờ
   &X-Amz-Signature=abc123...   ← chữ ký HMAC, không thể giả mạo
3. API trả URL này cho client
4. Client PUT file thẳng vào URL đó (MinIO verify chữ ký)
5. MinIO nhận file, lưu vào bucket
6. Client gọi /callback để API lưu metadata vào PostgreSQL
```

---

## 4. PostgreSQL (Metadata Database — thay thế AWS RDS)

### Là gì?
PostgreSQL là relational database mã nguồn mở, dùng để lưu **metadata** của file (thông tin mô tả file), không phải file thực sự.

### Lưu gì?

```sql
files table:
├── id            -- UUID, primary key
├── name          -- tên file hiển thị
├── original_name -- tên file gốc khi upload
├── mime_type     -- image/jpeg, application/pdf, ...
├── size          -- byte
├── bucket        -- "uploads" (tên MinIO bucket)
├── object_key    -- "uploads/2026/10/03/uuid_file.pdf" (vị trí trong MinIO)
├── status        -- uploaded | scanning | clean | infected | processing | ready | failed
├── checksum      -- SHA-256 hash của file
├── uploaded_by   -- user ID (từ auth header)
├── thumbnail_key -- object key của thumbnail (set bởi Thumbnail Worker)
├── ocr_text      -- extracted text (set bởi OCR Worker)
├── preview_key   -- object key của preview (set bởi Preview Worker)
├── created_at
└── updated_at
```

### Tại sao cần metadata DB khi đã có MinIO?

MinIO chỉ biết: "có file tên X ở bucket Y". Nó không biết:
- File này thuộc user nào?
- File đã được scan virus chưa?
- Checksum là gì?
- Đã tạo thumbnail chưa?

PostgreSQL lưu toàn bộ ngữ cảnh đó, cho phép query phức tạp:
```sql
SELECT * FROM files WHERE uploaded_by = 'user123' AND status = 'clean' AND mime_type = 'image/jpeg';
```

### Tại sao dùng Bun ORM thay vì raw SQL?

- **Bun** là ORM nhẹ cho Go, hỗ trợ PostgreSQL native, `bun.DB` wrap `*sql.DB` chuẩn.
- Có query builder type-safe, tránh SQL injection.
- Migration tách biệt bằng `golang-migrate` — không phụ thuộc ORM.

---

## 5. Apache Kafka (Message Queue — thay thế AWS SQS/SNS)

### Là gì?
Kafka là **distributed event streaming platform** — một hệ thống message queue có khả năng xử lý hàng triệu event/giây, lưu trữ event có thể replay được.

### Làm gì?
Khi file upload thành công, API phát một event vào Kafka topic `file.uploaded`:

```json
{
  "fileID": "550e8400-e29b-41d4-a716-446655440000",
  "bucket": "uploads",
  "objectKey": "uploads/2026/10/03/uuid_invoice.pdf",
  "mimeType": "application/pdf",
  "size": 1048576,
  "uploadedBy": "user123",
  "uploadedAt": "2026-10-03T14:00:00Z"
}
```

Các Workers lắng nghe topic này và xử lý độc lập.

### Tại sao cần Kafka? Tại sao không gọi trực tiếp từ API?

**Vấn đề nếu gọi trực tiếp (Synchronous):**

```
Client upload file
    → API nhận file
    → API gọi ClamAV scan (5-30 giây!)
    → API chờ... chờ... chờ...
    → API gọi Thumbnail generator (3-10 giây!)
    → API chờ... chờ... chờ...
    → API trả response cho client
```

Client phải chờ **30-60 giây** cho 1 request upload. Không thể chấp nhận.

**Giải pháp với Kafka (Asynchronous):**

```
Client upload file
    → API nhận file, lưu vào MinIO + PostgreSQL
    → API phát event vào Kafka (< 1ms)
    → API trả response ngay cho client: {"status": "uploaded", "id": "..."}

(Kafka background)
    → Virus Scan Worker nhận event → scan file → cập nhật DB
    → Thumbnail Worker nhận event → tạo thumbnail → upload MinIO → cập nhật DB
    → OCR Worker nhận event → extract text → cập nhật DB
```

Client nhận response trong **< 1 giây**. Workers xử lý nền không ảnh hưởng UX.

### Kafka KRaft Mode là gì?

Kafka truyền thống cần **ZooKeeper** để quản lý metadata cluster — đây là service phức tạp, tốn tài nguyên. **KRaft mode** (Kafka Raft) loại bỏ hoàn toàn ZooKeeper, Kafka tự quản lý metadata bằng Raft consensus protocol. Đơn giản hơn, ít container hơn, production-ready từ Kafka 3.3+.

---

## 6. Nginx (CDN / Reverse Proxy — thay thế AWS CloudFront / Cloudflare)

### Là gì?
Nginx là web server / reverse proxy mã nguồn mở. Trong hệ thống này, Nginx đóng vai trò **CDN nội bộ** và **reverse proxy**.

### Làm gì?

**Vai trò 1 — Reverse Proxy:**
```
Client → Nginx :80/:443 → File Upload API :8080
```
- Nginx là "cửa trước", ẩn API server khỏi internet.
- Xử lý SSL termination (HTTPS), client chỉ cần biết domain.
- Load balancing nếu scale nhiều instance API.

**Vai trò 2 — Static File Cache (thay CDN):**
```
Client GET /files/{FILE_ID}
    → Nginx kiểm tra cache
    ├─ Cache HIT  → trả file ngay từ Nginx (không cần gọi API)
    └─ Cache MISS → gọi MinIO lấy file → cache lại → trả cho client
```

> **Tại sao cần cache?** Nếu 1000 người cùng tải 1 ảnh, không có cache thì MinIO phải xử lý 1000 request. Với Nginx cache, chỉ có 1 request đến MinIO, 999 request còn lại được phục vụ từ Nginx (nhanh gấp 10-100x).

**Vai trò 3 — Rate Limiting (tránh DDoS):**
```nginx
limit_req_zone $binary_remote_addr zone=upload:10m rate=10r/s;
```
Giới hạn 10 request/giây mỗi IP — chặn client spam upload.

### Tại sao file public/static dùng CDN còn file private dùng Presigned URL?

| Loại file | Cơ chế | Lý do |
|---|---|---|
| Public (avatar, thumbnail) | Nginx cache / CDN | Ai cũng được tải, cần tốc độ cao |
| Private (tài liệu cá nhân) | Presigned URL (hết hạn) | Chỉ người được cấp phép mới tải được, URL hết hạn sau X phút |

---

## 7. Async Workers Pool

Workers là các Go service độc lập, chạy trong container riêng, lắng nghe Kafka và xử lý file sau khi upload.

### Worker 1 — Virus Scan (ClamAV)

**Làm gì?**
- Download file từ MinIO
- Gửi file cho ClamAV daemon scan
- Nếu clean → cập nhật `status = 'clean'` trong PostgreSQL
- Nếu infected → cập nhật `status = 'infected'`, gửi alert, optionally xóa file

**Tại sao cần?**

Nếu user upload virus lên hệ thống và hệ thống phục vụ file đó cho user khác → hệ thống trở thành phương tiện phát tán malware. ClamAV là antivirus engine mã nguồn mở, miễn phí, có virus database cập nhật liên tục.

**Tại sao scan async thay vì sync?**

ClamAV có thể mất 5-30 giây để scan file 100MB. Nếu block request upload → UX rất tệ. Scan async → user nhận response ngay, file được đánh dấu `scanning`, client poll `/files/{id}` để biết kết quả.

---

### Worker 2 — Thumbnail Generation (imaging / ffmpeg)

**Làm gì?**
- Download ảnh/video từ MinIO
- Resize/crop thành thumbnail (ví dụ 300x300px)
- Upload thumbnail lên MinIO với key: `thumbnails/{fileID}.jpg`
- Cập nhật `thumbnail_key` trong PostgreSQL

**Tại sao cần?**

- UI hiển thị danh sách file cần thumbnail để preview nhanh — không thể load cả file gốc 10MB chỉ để hiển thị ảnh nhỏ.
- Giảm bandwidth: thumbnail 5KB thay vì ảnh gốc 5MB.
- Dùng `github.com/disintegration/imaging` cho ảnh, `ffmpeg` cho video (extract frame đầu tiên).

---

### Worker 3 — OCR (Tesseract)

**Làm gì?**
- Download file PDF / ảnh từ MinIO
- Gọi Tesseract OCR để extract text
- Lưu text vào PostgreSQL field `ocr_text`

**Tại sao cần?**

- Cho phép **full-text search** trên nội dung file: "Tìm tất cả hợp đồng có tên 'Nguyễn Văn A'".
- Document management systems (DMS) cần tính năng này để phân loại tài liệu tự động.
- Tesseract là OCR engine mã nguồn mở của Google, hỗ trợ 100+ ngôn ngữ.

---

### Worker 4 — Preview Generation (LibreOffice / ffmpeg)

**Làm gì?**
- **PDF/Word/Excel** → Convert sang ảnh/HTML preview bằng LibreOffice headless mode
- **Video** → Extract frame + tạo preview GIF bằng ffmpeg
- Upload preview lên MinIO: `previews/{fileID}/page-1.jpg`
- Cập nhật `preview_key` trong PostgreSQL

**Tại sao cần?**

User muốn preview tài liệu trên browser mà không cần download về máy. Đặc biệt quan trọng với:
- Word/Excel/PowerPoint (browser không render được native)
- Video (cần thumbnail và clip preview)

---

### Worker 5 — Notification (SMTP / WebSocket)

**Làm gì?**
- Lắng nghe các event: `file.uploaded`, `file.scanned`, `file.thumbnail.done`
- Gửi email notification qua SMTP: "File của bạn đã upload thành công và đã được scan sạch"
- Push realtime notification qua WebSocket: client nhận ngay khi scan xong mà không cần F5

**Tại sao cần?**

Với async processing, user không biết khi nào file sẵn sàng sử dụng. Notification giải quyết vấn đề này:
- **Email**: thông báo kết quả quan trọng (file bị infected, file quá lớn không xử lý được)
- **WebSocket**: UX realtime — thanh progress bar, icon "đang xử lý" chuyển sang "sẵn sàng"

---

## 8. Luồng Dữ Liệu Chi Tiết

### Luồng 1: Direct Upload (file nhỏ ≤ 100MB)

```
1. Client   POST /api/v1/files/upload (multipart/form-data)
             └─ Header: X-User-ID: user123

2. API       Validate request:
             ├─ Giới hạn body 100MB (MaxBytesReader)
             ├─ Parse multipart form (32MB RAM, còn lại ra disk)
             ├─ Đọc 512 byte đầu → detect MIME
             └─ Kiểm tra MIME trong AllowedMIMETypes

3. API       Tạo object key:
             └─ uploads/2026/10/03/{uuid}_{filename}.pdf

4. API       Upload lên MinIO (streaming, không buffer toàn bộ):
             └─ TeeReader: đọc 1 lần, ghi vào MinIO VÀ SHA256 hasher đồng thời

5. API       Lưu metadata vào PostgreSQL:
             └─ INSERT INTO files (id, name, mime_type, size, bucket, object_key,
                                   status, checksum, uploaded_by, ...) VALUES (...)

6. API       Phát event vào Kafka topic 'file.uploaded':
             └─ {fileID, bucket, objectKey, mimeType, size, uploadedBy}

7. API       Trả response 201 Created:
             └─ {"id": "...", "name": "...", "status": "uploaded", "checksum": "..."}

8. (Async)   Kafka Workers nhận event, xử lý song song:
             ├─ Virus Scan Worker → update status
             ├─ Thumbnail Worker → tạo thumbnail
             └─ OCR Worker → extract text
```

### Luồng 2: Presigned Upload (file lớn > 100MB)

```
1. Client   POST /api/v1/files/presigned-upload
             └─ Body: {"filename": "video.mp4", "size": 500000000, "mimeType": "video/mp4"}

2. API       Validate filename, size, mimeType
             Tạo object key: uploads/2026/10/03/{uuid}_video.mp4
             Gọi MinIO SDK: PresignedPutObject(bucket, key, expiry=1h)
             └─ MinIO trả về URL có chữ ký, hết hạn sau 1 giờ

3. API       Trả response:
             └─ {"fileID": "...", "uploadURL": "https://minio:9000/bucket/...?signature=...", "expiresIn": 3600}

4. Client   PUT {uploadURL} (upload thẳng vào MinIO, không qua API)
             └─ MinIO verify chữ ký → lưu file → trả 200 OK

5. Client   POST /api/v1/files/callback
             └─ Body: {"fileID": "...", "objectKey": "..."}

6. API       Lưu metadata vào PostgreSQL
             Phát event vào Kafka 'file.uploaded'
             Trả response 201 Created

7. (Async)   Workers xử lý như luồng 1
```

### Luồng 3: Presigned Download (file private)

```
1. Client   GET /api/v1/files/{id}/presigned-download

2. API       Query PostgreSQL → lấy objectKey, kiểm tra quyền truy cập (Sprint 11)
             Gọi MinIO SDK: PresignedGetObject(bucket, key, expiry=15m)
             └─ MinIO trả URL có chữ ký, hết hạn 15 phút

3. API       Trả response:
             └─ {"downloadURL": "https://minio:9000/...?signature=...", "expiresIn": 900}

4. Client   GET {downloadURL} (download thẳng từ MinIO)
             └─ MinIO verify chữ ký → stream file → không qua API
```

---

## 9. Tại Sao Thiết Kế Này Có Thể Scale?

| Bottleneck | Giải pháp |
|---|---|
| API server quá tải | Scale ngang (nhiều container) + Nginx load balancing |
| MinIO dung lượng đầy | Thêm node MinIO (distributed mode) |
| PostgreSQL chậm | Read replicas cho query; connection pooling (PgBouncer) |
| Kafka lag (worker xử lý không kịp) | Tăng số partition, thêm worker instance |
| Upload lớn làm nghẽn API | Presigned URL: client upload thẳng vào MinIO |
| Static file chậm | Nginx cache: file phổ biến không cần đến MinIO |

---

## 10. Lựa Chọn Công Nghệ — Câu Hỏi Thường Gặp

**Q: Tại sao Kafka thay vì RabbitMQ?**
Kafka lưu event có thể **replay** — nếu Virus Scan Worker crash, sau khi restart nó đọc lại event từ Kafka và xử lý tiếp. RabbitMQ xóa message sau khi consumer ACK, không replay được.

**Q: Tại sao MinIO thay vì Ceph?**
Ceph rất mạnh nhưng phức tạp, cần cluster nhiều node. MinIO đơn giản hơn, single binary, API S3-compatible, phù hợp cho dự án vừa và nhỏ. Dễ migrate lên AWS S3 thật khi cần.

**Q: Tại sao Go thay vì Node.js / Python?**
- Go compile ra binary tĩnh, không cần runtime, deploy đơn giản.
- Goroutine xử lý concurrency tốt hơn thread model của Node.js/Python cho I/O-heavy workload như file upload.
- Memory footprint thấp hơn Node.js đáng kể.

**Q: Tại sao không dùng Gin/Fiber mà dùng `net/http`?**
Go 1.22+ đã hỗ trợ pattern matching trong routing (`GET /files/{id}`), đủ dùng cho project này. Giảm dependency, dễ học nguyên lý HTTP server thực sự.

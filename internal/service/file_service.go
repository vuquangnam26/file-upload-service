package service

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"file-upload-service/internal/domain"
	"file-upload-service/internal/queue"
	"file-upload-service/internal/repository"
	"file-upload-service/internal/storage"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	MaxFileSize = 100 * 1024 * 1024 // 100 MB
)

// AllowedMIMETypes danh sách MIME type được phép upload
var AllowedMIMETypes = map[string]bool{
	"image/jpeg":      true,
	"image/png":       true,
	"image/gif":       true,
	"image/webp":      true,
	"application/pdf": true,
	"video/mp4":       true,
}

type UploadInput struct {
	Reader     io.Reader
	Filename   string
	Size       int64
	UploadedBy string
}
type UploadResult struct {
	File *domain.File
}
type FileService interface {
	Upload(ctx context.Context, input UploadInput) (*UploadResult, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.File, error)
}
type fileService struct {
	storage  storage.Storage
	repo     repository.FileRepository
	bucket   string
	producer queue.EventProducer
	topic    string
}

// GetByID implements [FileService].
func (s *fileService) GetByID(ctx context.Context, id uuid.UUID) (*domain.File, error) {
	return s.repo.GetByID(ctx, id)
}

// Upload implements [FileService].
func (s *fileService) Upload(ctx context.Context, input UploadInput) (*UploadResult, error) {
	if input.Size > MaxFileSize {
		return nil, fmt.Errorf("file too large: max %d MB", MaxFileSize/1024/1024)
	}
	// 2. Đọc 512 byte đầu để detect MIME bằng Magic Bytes
	//    (KHÔNG tin vào Content-Type header của client)

	buf := make([]byte, 512)
	n, err := input.Reader.Read(buf)
	if err != nil && err != io.EOF {
		return nil, fmt.Errorf("failed to read file header: %w", err)
	}
	detectedMIME := http.DetectContentType(buf[:n])
	// Strip params ví dụ "text/plain; charset=utf-8" → "text/plain"
	mimeType, _, _ := mime.ParseMediaType(detectedMIME)
	if !AllowedMIMETypes[mimeType] {
		return nil, fmt.Errorf("file type not allowed: %s", mimeType)
	}
	// 3. Reconstruct reader: ghép 512 byte đã đọc + phần còn lại
	//fullReader := io.MultiReader(strings.NewReader(string(buf[:n])), input.Reader)
	fullReader := io.MultiReader(bytes.NewReader(buf[:n]), input.Reader)
	// 4. Tính Checksum SHA-256 song song với upload
	//    Dùng TeeReader để đọc 1 lần nhưng ghi vào cả MinIO và hasher
	hasher := sha256.New()
	tee := io.TeeReader(fullReader, hasher)
	// 5. Tạo Object Key theo pattern: uploads/{yyyy}/{mm}/{dd}/{uuid}_{filename}
	fileID := uuid.New()
	now := time.Now()
	ext := getExtension(input.Filename)
	objectKey := fmt.Sprintf("uploads/%d/%02d/%02d/%s_%s%s",
		now.Year(), now.Month(), now.Day(),
		fileID.String(),
		sanitizeFilename(input.Filename),
		ext,
	)
	// 6. Upload lên MinIO
	if err := s.storage.Upload(ctx, objectKey, tee, -1, mimeType); err != nil {
		return nil, fmt.Errorf("failed to upload to storage: %w", err)
	}

	// 7. Tính checksum sau khi upload xong
	checksum := hex.EncodeToString(hasher.Sum(nil))
	// 8. Lưu metadata vào PostgreSQL
	file := &domain.File{
		ID:           fileID,
		Name:         input.Filename,
		OriginalName: input.Filename,
		MimeType:     mimeType,
		Size:         input.Size,
		Bucket:       s.bucket,
		ObjectKey:    objectKey,
		Status:       domain.FileStatusUploaded,
		Checksum:     checksum,
		UploadedBy:   input.UploadedBy,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := s.repo.Create(ctx, file); err != nil {
		// TODO Sprint 5: Rollback — xóa file khỏi MinIO nếu lưu DB thất bại
		return nil, fmt.Errorf("failed to save file metadata: %w", err)
	}
	// 9. Publish event "file.uploaded" lên Kafka
	event := domain.FileUploadedEvent{
		FileID:    file.ID,
		Bucket:    file.Bucket,
		ObjectKey: file.ObjectKey,
		MimeType:  file.MimeType,
		Size:      file.Size,
		Checksum:  file.Checksum,
		Timestamp: now,
	}
	if err := s.producer.Publish(ctx, s.topic, file.ID.String(), event); err != nil {
		// Log warning nhưng KHÔNG return error — upload vẫn thành công
		log.Printf("[FileService] WARNING: failed to publish file.uploaded event: %v", err)
	}
	return &UploadResult{File: file}, nil
}

// sanitizeFilename loại bỏ ký tự nguy hiểm trong tên file

func sanitizeFilename(name string) string {
	base := strings.TrimSuffix(name, getExtension(name)) // ← Strip ext ra trước
	replacer := strings.NewReplacer(" ", "_", "/", "_", "\\", "_", "..", "_")
	return replacer.Replace(base)
}

// getExtension trả về extension có dấu chấm: ".jpg", ".pdf"
func getExtension(filename string) string {
	for i := len(filename) - 1; i >= 0; i-- {
		if filename[i] == '.' {
			return filename[i:]
		}
	}
	return ""
}

func NewFileService(
	s storage.Storage,
	r repository.FileRepository,
	bucket string,
	producer queue.EventProducer,
	topic string,
) FileService {
	return &fileService{
		storage:  s,
		repo:     r,
		bucket:   bucket,
		producer: producer,
		topic:    topic,
	}
}

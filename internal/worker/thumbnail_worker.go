package worker

import (
	"context"
	"file-upload-service/internal/domain"
	"file-upload-service/internal/queue"
	"file-upload-service/internal/repository"
	"file-upload-service/internal/storage"
	"file-upload-service/internal/thumbnail"
	"fmt"
	"log"
	"strings"
	"time"
)

type ThumbnailWorker struct {
	storage   storage.Storage
	repo      repository.FileRepository
	generator thumbnail.Generator
	producer  queue.EventProducer
	doneTopic string
}

func NewThumbnailWorker(
	s storage.Storage,
	r repository.FileRepository,
	g thumbnail.Generator,
	p queue.EventProducer,
	doneTopic string,
) *ThumbnailWorker {
	return &ThumbnailWorker{
		storage:   s,
		repo:      r,
		generator: g,
		producer:  p,
		doneTopic: doneTopic,
	}
}
func (w *ThumbnailWorker) ProcessEvent(ctx context.Context, topic string, key []byte, value []byte) error {
	event, err := queue.UnmarshalEvent[domain.FileUploadedEvent](value)
	if err != nil {
		return fmt.Errorf("unmarshal FileUploadedEvent: %w", err)
	}
	// Only process image files
	if !strings.HasPrefix(event.MimeType, "image/") {
		log.Printf("[ThumbnailWorker] Skipping non-image file: %s (mime=%s)", event.FileID, event.MimeType)
		return nil
	}
	log.Printf("[ThumbnailWorker] Generating thumbnail for fileID=%s", event.FileID)
	// 1. Download original file from MinIO
	stream, err := w.storage.Download(ctx, event.ObjectKey)
	if err != nil {
		return fmt.Errorf("download from storage: %w", err)
	}
	defer stream.Close()
	// 2. Generate thumbnail
	result, err := w.generator.Generate(ctx, stream, event.MimeType)
	if err != nil {
		return fmt.Errorf("generate thumbnail: %w", err)
	}
	// 3. Upload thumbnail to MinIO
	thumbKey := fmt.Sprintf("thumbnails/%s.jpg", event.FileID)
	if err := w.storage.Upload(ctx, thumbKey, result.Reader, result.Size, result.ContentType); err != nil {
		return fmt.Errorf("upload thumbnail: %w", err)
	}
	// 4. Update database
	if err := w.repo.UpdateThumbnailKey(ctx, event.FileID, thumbKey); err != nil {
		return fmt.Errorf("update thumbnail key: %w", err)
	}
	log.Printf("[ThumbnailWorker] Thumbnail created for fileID=%s key=%s", event.FileID, thumbKey)
	// 5. Publish file.thumbnail.done event
	doneEvent := domain.FileThumbnailDoneEvent{
		FileID:       event.FileID,
		ThumbnailKey: thumbKey,
		Timestamp:    time.Now(),
	}
	if err := w.producer.Publish(ctx, w.doneTopic, event.FileID.String(), doneEvent); err != nil {
		log.Printf("[ThumbnailWorker] Warning: failed to publish thumbnail.done event: %v", err)
	}
	return nil
}

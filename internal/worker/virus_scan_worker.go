package worker

import (
	"context"
	"file-upload-service/internal/domain"
	"file-upload-service/internal/queue"
	"file-upload-service/internal/repository"
	"file-upload-service/internal/scanner"
	"file-upload-service/internal/storage"
	"fmt"
	"log"
	"time"
)

type VirusScanWorker struct {
	storage      storage.Storage
	repo         repository.FileRepository
	scanner      scanner.VirusScanner
	producer     queue.EventProducer
	scannedTopic string
}

func NewVirusScanWorker(
	s storage.Storage,
	r repository.FileRepository,
	sc scanner.VirusScanner,
	p queue.EventProducer,
	scannedTopic string,
) *VirusScanWorker {
	return &VirusScanWorker{
		storage:      s,
		repo:         r,
		scanner:      sc,
		producer:     p,
		scannedTopic: scannedTopic,
	}
}

// ProcessEvent là callback hàm MessageHandler cho Kafka Consumer
func (w *VirusScanWorker) ProcessEvent(ctx context.Context, topic string, key []byte, value []byte) error {
	// 1. Unmarshal event từ Kafka
	event, err := queue.UnmarshalEvent[domain.FileUploadedEvent](value)
	if err != nil {
		return fmt.Errorf("unmarshal FileUploadedEvent: %w", err)
	}
	log.Printf("[VirusScanWorker] Start scanning fileID=%s key=%s", event.FileID, event.ObjectKey)
	// 2. Cập nhật status -> scanning
	if err := w.repo.UpdateStatus(ctx, event.FileID, domain.FileStatusScanning); err != nil {
		log.Printf("[VirusScanWorker] Failed to update status to scanning: %v", err)
	}
	// 3. Download stream từ MinIO
	stream, err := w.storage.Download(ctx, event.ObjectKey)
	if err != nil {
		_ = w.repo.UpdateStatus(ctx, event.FileID, domain.FileStatusFailed)
		return fmt.Errorf("failed to download from storage: %w", err)
	}
	defer stream.Close()
	// 4. Scan qua ClamAV
	result, err := w.scanner.Scan(ctx, stream)
	if err != nil {
		_ = w.repo.UpdateStatus(ctx, event.FileID, domain.FileStatusFailed)
		return fmt.Errorf("failed to scan file: %w", err)
	}
	// 5. Cập nhật DB dựa trên kết quả
	finalStatus := domain.FileStatusClean
	if !result.IsClean {
		finalStatus = domain.FileStatusInfected
		log.Printf("[VirusScanWorker] ⚠️ ALERT: Infected file detected! fileID=%s virus=%s", event.FileID, result.VirusName)
	} else {
		log.Printf("[VirusScanWorker] ✅ File is clean! fileID=%s", event.FileID)
	}
	if err := w.repo.UpdateStatus(ctx, event.FileID, finalStatus); err != nil {
		return fmt.Errorf("failed to update final status: %w", err)
	}
	// 6. Phát event file.scanned lên Kafka
	scannedEvent := domain.FileScannedEvent{
		FileID:    event.FileID,
		Status:    finalStatus,
		VirusName: result.VirusName,
		Timestamp: time.Now(),
	}
	if err := w.producer.Publish(ctx, w.scannedTopic, event.FileID.String(), scannedEvent); err != nil {
		log.Printf("[VirusScanWorker] Warning: failed to publish file.scanned event: %v", err)
	}
	return nil
}

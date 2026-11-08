package domain

import (
	"time"

	"github.com/google/uuid"
)

type FileUploadedEvent struct {
	FileID    uuid.UUID `json:"file_id"`
	Bucket    string    `json:"bucket"`
	ObjectKey string    `json:"object_key"`
	MimeType  string    `json:"mime_type"`
	Size      int64     `json:"size"`
	Checksum  string    `json:"checksum"`
	Timestamp time.Time `json:"timestamp"`
}

type FileScannedEvent struct {
	FileID    uuid.UUID  `json:"file_id"`
	Status    FileStatus `json:"status"` // "clean" or "infected"
	VirusName string     `json:"virus_name,omitempty"`
	Timestamp time.Time  `json:"timestamp"`
}
type FileThumbnailDoneEvent struct {
    FileID       uuid.UUID `json:"file_id"`
    ThumbnailKey string    `json:"thumbnail_key"`
    Timestamp    time.Time `json:"timestamp"`
}
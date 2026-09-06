package domain

import (
	"time"

	"github.com/google/uuid" // ← fix import
)

// FileStatus represents the processing state of an uploaded file.
type FileStatus string

const (
	FileStatusUploaded FileStatus = "uploaded"
	FileStatusScanning FileStatus = "scanning"
	FileStatusClean    FileStatus = "clean"
	FileStatusInfected FileStatus = "infected"
	FileStatusFailed   FileStatus = "failed"
)

type File struct {
	ID           uuid.UUID
	Name         string
	OriginalName string
	MimeType     string
	Size         int64
	Bucket       string
	ObjectKey    string
	Status       FileStatus
	Checksum     string
	UploadedBy   string
	CreatedAt    time.Time
	UpdatedAt    time.Time
	DeletedAt    *time.Time
}

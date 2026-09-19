package domain

import (
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type FileStatus string

const (
	FileStatusUploaded FileStatus = "uploaded"
	FileStatusScanning FileStatus = "scanning"
	FileStatusClean    FileStatus = "clean"
	FileStatusInfected FileStatus = "infected"
	FileStatusFailed   FileStatus = "failed"
)

type File struct {
	bun.BaseModel `bun:"table:files,alias:f"` // ← Thêm dòng này để Bun biết tên bảng

	ID           uuid.UUID  `bun:"id,pk,type:uuid"       json:"id"`
	Name         string     `bun:"name,notnull"          json:"name"`
	OriginalName string     `bun:"original_name,notnull" json:"original_name"`
	MimeType     string     `bun:"mime_type,notnull"     json:"mime_type"`
	Size         int64      `bun:"size,notnull"          json:"size"`
	Bucket       string     `bun:"bucket,notnull"        json:"bucket"`
	ObjectKey    string     `bun:"object_key,notnull"    json:"object_key"`
	Status       FileStatus `bun:"status,notnull"        json:"status"`
	Checksum     string     `bun:"checksum"              json:"checksum,omitempty"`
	UploadedBy   string     `bun:"uploaded_by"           json:"uploaded_by,omitempty"`
	CreatedAt    time.Time  `bun:"created_at,notnull"    json:"created_at"`
	UpdatedAt    time.Time  `bun:"updated_at,notnull"    json:"updated_at"`
	DeletedAt    *time.Time `bun:"deleted_at,soft_delete" json:"-"`
}

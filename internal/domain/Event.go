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
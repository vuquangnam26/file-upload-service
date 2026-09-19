package storage

import (
	"context"
	"io"
	"time"
)

type Storage interface {
	Upload(ctx context.Context, key string, reader io.Reader, size int64, contentType string) error
	Download(ctx context.Context, key string) (io.ReadCloser, error)
	GetPresignedUploadURL(ctx context.Context, key string, expiry time.Duration) (string, error)
	GetPresignedDownloadURL(ctx context.Context, key string, expiry time.Duration) (string, error)
	Delete(ctx context.Context, key string) error
}

package thumbnail

import (
	"context"
	"io"
)

type GenerateResult struct {
	Reader      io.Reader
	Size        int64
	ContentType string // "image/jpeg"
}
type Generator interface {
	Generate(ctx context.Context, r io.Reader, mimeType string) (*GenerateResult, error)
}

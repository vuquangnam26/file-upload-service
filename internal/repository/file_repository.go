package repository

import (
	"context"
	"file-upload-service/internal/domain"

	"github.com/google/uuid"
)

type FileRepository interface {
	Create(ctx context.Context, file *domain.File) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.File, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status domain.FileStatus) error
	SoftDelete(ctx context.Context, id uuid.UUID) error
}

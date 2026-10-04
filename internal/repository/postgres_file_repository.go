package repository

import (
	"context"
	"file-upload-service/internal/domain"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type postgresFileRepository struct {
	db *bun.DB
}

func NewPostgresFileRepository(db *bun.DB) FileRepository {
	return &postgresFileRepository{db: db}
}
func (r *postgresFileRepository) Create(ctx context.Context, file *domain.File) error {
	_, err := r.db.NewInsert().Model(file).Exec(ctx)
	if err != nil {
		return fmt.Errorf("repository.Create: %w", err)
	}
	return nil
}
func (r *postgresFileRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.File, error) {
	file := &domain.File{}
	err := r.db.NewSelect().
		Model(file).
		Where("id = ?", id).
		Where("deleted_at IS NULL"). // soft delete filter
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("repository.GetByID: %w", err)
	}
	return file, nil
}
func (r *postgresFileRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.FileStatus) error {
	_, err := r.db.NewUpdate().
		TableExpr("files").
		Set("status = ?", status).
		Set("updated_at = ?", time.Now()).
		Where("id = ?", id).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("repository.UpdateStatus: %w", err)
	}
	return nil
}
func (r *postgresFileRepository) SoftDelete(ctx context.Context, id uuid.UUID) error {
	now := time.Now()
	_, err := r.db.NewUpdate().
		TableExpr("files").
		Set("deleted_at = ?", now).
		Set("updated_at = ?", now).
		Where("id = ?", id).
		Where("deleted_at IS NULL").
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("repository.SoftDelete: %w", err)
	}
	return nil
}

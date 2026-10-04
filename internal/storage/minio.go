package storage

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"time"

	"github.com/minio/minio-go/v7"
)

type minioStorage struct {
	client *minio.Client
	bucket string
}

func NewMinIOStorage(client *minio.Client, bucket string) Storage {
	return &minioStorage{
		client: client,
		bucket: bucket,
	}

}
func (s *minioStorage) Upload(ctx context.Context, key string, reader io.Reader, size int64, contentType string) error {
	_, err := s.client.PutObject(ctx, s.bucket, key, reader, size, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return fmt.Errorf("failed to upload object to minio: %w", err)
	}
	return nil
}
func (s *minioStorage) Download(ctx context.Context, key string) (io.ReadCloser, error) {
	obj, err := s.client.GetObject(ctx, s.bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get object from minio: %w", err)
	}
	return obj, nil
}
func (s *minioStorage) GetPresignedUploadURL(ctx context.Context, key string, expiry time.Duration) (string, error) {
	presignedURL, err := s.client.PresignedPutObject(ctx, s.bucket, key, expiry)
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned upload url: %w", err)
	}
	return presignedURL.String(), nil
}
func (s *minioStorage) GetPresignedDownloadURL(ctx context.Context, key string, expiry time.Duration) (string, error) {
	reqParams := make(url.Values)
	presignedURL, err := s.client.PresignedGetObject(ctx, s.bucket, key, expiry, reqParams)
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned download url: %w", err)
	}
	return presignedURL.String(), nil
}
func (s *minioStorage) Delete(ctx context.Context, key string) error {
	err := s.client.RemoveObject(ctx, s.bucket, key, minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete object from minio: %w", err)
	}
	return nil
}

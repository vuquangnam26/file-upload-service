package adapter

import "context"

// MinIOPinger - Adapter interface for MinIO client.
// Prevents leaking minio-go SDK dependency directly into health check layer.
type MinIOPinger interface {
	BucketExists(ctx context.Context, bucketName string) (bool, error)
}

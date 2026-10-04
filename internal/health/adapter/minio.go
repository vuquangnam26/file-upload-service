package adapter

import "context"

// MinIOPinger — Adapter interface cho MinIO client.
// Tránh leak dependency minio-go SDK vào layer health check.
type MinIOPinger interface {
	BucketExists(ctx context.Context, bucketName string) (bool, error)
}

package health

import (
	"context"
	"file-upload-service/internal/health/adapter"
	"fmt"
	"time"
)

type MinIOHealthCheck struct {
	pinger adapter.MinIOPinger
	bucket string
}

func NewMinIOHealthCheck(pinger adapter.MinIOPinger, bucket string) *MinIOHealthCheck {
	return &MinIOHealthCheck{
		pinger: pinger,
		bucket: bucket,
	}
}

func (m *MinIOHealthCheck) Name() string {
	return "minio"
}

func (c *MinIOHealthCheck) Check(timeout time.Duration) Result {
	start := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	exists, err := c.pinger.BucketExists(ctx, c.bucket)
	if err != nil {
		return Result{Status: StatusFail, Error: err.Error()}
	}
	if !exists {
		return Result{Status: StatusDegre, Error: fmt.Sprintf("bucket %q not found", c.bucket)}
	}
	return Result{
		Status:    StatusOk,
		LatencyMs: time.Since(start).Milliseconds(),
	}
}

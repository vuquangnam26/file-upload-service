package health

import (
	"context"
	"file-upload-service/internal/health/adapter"
	"time"
)

type KafkaHealthCheck struct {
	pinger adapter.KafkaPinger
}

func NewKafkaHealthCheck(pinger adapter.KafkaPinger) *KafkaHealthCheck {
	return &KafkaHealthCheck{pinger: pinger}
}

func (c *KafkaHealthCheck) Name() string {
	return "kafka"
}
func (c *KafkaHealthCheck) Check(timeout time.Duration) Result {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	start := time.Now()
	if err := c.pinger.Ping(ctx); err != nil {
		return Result{
			Status: StatusFail,
			Error:  err.Error(),
		}
	}
	return Result{
		Status:    StatusOk,
		LatencyMs: time.Since(start).Milliseconds(),
	}
}

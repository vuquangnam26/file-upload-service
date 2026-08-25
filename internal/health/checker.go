package health

import "time"

type Status string

const (
	StatusOk    Status = "ok"
	StatusDegre Status = "degraded"
	StatusFail  Status = "fail"
)

type Result struct {
	Status    Status `json:"status"`
	LatencyMs int64  `json:"latency_ms,omitempty"`
	Error     string `json:"error,omitempty"`
}

type Report struct {
	Status  Status            `json:"status"`
	Checkes map[string]Result `json:"checks"`
}

// Checker — Strategy interface, mỗi service implement cái này
type Checker interface {
	Name() string
	Check(timeout time.Duration) Result
}

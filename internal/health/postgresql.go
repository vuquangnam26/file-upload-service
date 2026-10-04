package health

import (
	"context"
	"file-upload-service/bootstrap/database"
	"time"
)

type PostgreSQLHealthCheck struct {
	db database.Database
}

func NewPostgreSQLHealthCheck(db database.Database) *PostgreSQLHealthCheck {
	return &PostgreSQLHealthCheck{db: db}
}
func (c *PostgreSQLHealthCheck) Name() string { return "postgresql" }

func (c *PostgreSQLHealthCheck) Check(timeout time.Duration) Result {
	start := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	_ = ctx
	if err := c.db.Ping(); err != nil {
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

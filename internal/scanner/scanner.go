package scanner

import (
	"context"
	"io"
)

type ScanResult struct {
	IsClean   bool
	VirusName string
}

type VirusScanner interface {
	Scan(ctx context.Context, r io.Reader) (*ScanResult, error)
	Ping(ctx context.Context) error
}

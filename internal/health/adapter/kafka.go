package adapter

import "context"

type KafkaPinger interface {
	Ping(ctx context.Context) error
}

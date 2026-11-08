package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/twmb/franz-go/pkg/kgo"
)

type MessageHandler func(ctx context.Context, topic string, key []byte, value []byte) error

// EventConsumer manages the Kafka message polling loop.
// (Single Responsibility: handles polling and dispatching, independent of business logic)
type EventConsumer struct {
	client  *kgo.Client
	handler MessageHandler
}

func NewKafkaConsumer(brokers []string, groupID string, topics []string, handler MessageHandler) (*EventConsumer, error) {
	client, err := kgo.NewClient(
		kgo.SeedBrokers(brokers...),
		kgo.ConsumerGroup(groupID),
		kgo.ConsumeTopics(topics...),
		// Start reading from the earliest uncommitted message for new groups
		kgo.ConsumeResetOffset(kgo.NewOffset().AtStart()),
	)
	if err != nil {
		return nil, fmt.Errorf("queue.NewKafkaConsumer: %w", err)
	}
	return &EventConsumer{
		client:  client,
		handler: handler,
	}, nil
}

// Start begins the polling loop. Blocks until ctx is cancelled (graceful shutdown).
func (c *EventConsumer) Start(ctx context.Context) {
	log.Println("[Consumer] Starting consumer loop...")
	for {
		fetches := c.client.PollFetches(ctx)
		// If context is cancelled (shutdown signal) -> exit loop
		if ctx.Err() != nil {
			log.Println("[Consumer] Context cancelled, stopping consumer loop")
			break
		}
		// Log fetch errors if any (network, broker down, etc.)
		if errs := fetches.Errors(); len(errs) > 0 {
			for _, e := range errs {
				log.Printf("[Consumer] Fetch error topic=%s partition=%d: %v", e.Topic, e.Partition, e.Err)
			}
			continue
		}
		// Process each record
		fetches.EachRecord(func(record *kgo.Record) {
			if err := c.handler(ctx, record.Topic, record.Key, record.Value); err != nil {
				log.Printf("[Consumer] Handler error topic=%s key=%s: %v",
					record.Topic, string(record.Key), err)
				// TODO: Implement retry / dead-letter queue
			}
		})
		// Commit offsets after processing batch
		if err := c.client.CommitUncommittedOffsets(ctx); err != nil {
			log.Printf("[Consumer] Commit error: %v", err)
		}
	}
}
func (c *EventConsumer) Close() {
	c.client.Close()
}

// UnmarshalEvent is a helper for workers to unmarshal JSON into a specific struct.
func UnmarshalEvent[T any](value []byte) (T, error) {
	var event T
	if err := json.Unmarshal(value, &event); err != nil {
		return event, fmt.Errorf("queue.UnmarshalEvent: %w", err)
	}
	return event, nil
}

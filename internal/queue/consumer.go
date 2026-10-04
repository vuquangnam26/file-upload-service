package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/twmb/franz-go/pkg/kgo"
)

type MessageHandler func(ctx context.Context, topic string, key []byte, value []byte) error

// EventConsumer quản lý vòng lặp đọc message từ Kafka.
// (Single Responsibility: chỉ lo polling + dispatch, không biết logic xử lý)
type EventConsumer struct {
	client  *kgo.Client
	handler MessageHandler
}

func NewKafkaConsumer(brokers []string, groupID string, topics []string, handler MessageHandler) (*EventConsumer, error) {
	client, err := kgo.NewClient(
		kgo.SeedBrokers(brokers...),
		kgo.ConsumerGroup(groupID),
		kgo.ConsumeTopics(topics...),
		// Bắt đầu đọc từ message mới nhất chưa commit nếu group mới
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

// Start bắt đầu vòng lặp polling. Block cho đến khi ctx bị cancel (graceful shutdown).
func (c *EventConsumer) Start(ctx context.Context) {
	log.Println("[Consumer] Starting consumer loop...")
	for {
		fetches := c.client.PollFetches(ctx)
		// Nếu context bị cancel (shutdown signal) → thoát vòng lặp
		if ctx.Err() != nil {
			log.Println("[Consumer] Context cancelled, stopping consumer loop")
			break
		}
		// Log lỗi fetch nếu có (network, broker down, v.v.)
		if errs := fetches.Errors(); len(errs) > 0 {
			for _, e := range errs {
				log.Printf("[Consumer] Fetch error topic=%s partition=%d: %v", e.Topic, e.Partition, e.Err)
			}
			continue
		}
		// Xử lý từng record
		fetches.EachRecord(func(record *kgo.Record) {
			if err := c.handler(ctx, record.Topic, record.Key, record.Value); err != nil {
				log.Printf("[Consumer] Handler error topic=%s key=%s: %v",
					record.Topic, string(record.Key), err)
				// TODO: Implement retry / dead-letter queue
			}
		})
		// Auto-commit offsets sau khi xử lý xong batch
		if err := c.client.CommitUncommittedOffsets(ctx); err != nil {
			log.Printf("[Consumer] Commit error: %v", err)
		}
	}
}
func (c *EventConsumer) Close() {
	c.client.Close()
}

// UnmarshalEvent là helper để Worker unmarshal JSON vào struct cụ thể.
func UnmarshalEvent[T any](value []byte) (T, error) {
	var event T
	if err := json.Unmarshal(value, &event); err != nil {
		return event, fmt.Errorf("queue.UnmarshalEvent: %w", err)
	}
	return event, nil
}

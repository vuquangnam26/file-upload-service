package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/twmb/franz-go/pkg/kgo"
)

type EventProducer interface {
	Publish(ctx context.Context, topic string, key string, payload any) error
	Close()
}

// kafkaProducer implement EventProducer bằng franz-go.
// (Single Responsibility: chỉ lo việc đẩy message vào Kafka)
type kafkaProducer struct {
	client *kgo.Client
}

func NewKafkaProducer(client *kgo.Client) EventProducer {
	return &kafkaProducer{client: client}
}

func (p *kafkaProducer) Publish(ctx context.Context, topic string, key string, payload any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("queue.Publish: failed to marshal payload: %w", err)
	}
	record := &kgo.Record{
		Topic: topic,
		Key:   []byte(key),
		Value: data,
	}
	// ProduceSync đợi Kafka ACK trước khi trả về — đảm bảo message không bị mất.
	results := p.client.ProduceSync(ctx, record)
	if err := results.FirstErr(); err != nil {
		return fmt.Errorf("queue.Publish: failed to produce to topic %s: %w", topic, err)
	}
	return nil
}
func (p *kafkaProducer) Close() {
	p.client.Close()
}
package driver

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/segmentio/kafka-go"

	"github.com/arklib/ark/queue"
)

type KafkaDriver struct {
	queue.Driver
	brokers []string
	Writer  *kafka.Writer
}

func NewKafkaDriver(brokers ...string) *KafkaDriver {
	writer := &kafka.Writer{
		Addr:                   kafka.TCP(brokers...),
		Balancer:               &kafka.Hash{},
		AllowAutoTopicCreation: true,
		BatchSize:              1,
	}
	return &KafkaDriver{
		Writer:  writer,
		brokers: brokers,
	}
}

func (k *KafkaDriver) Produce(ctx context.Context, topic string, rawMessage []byte) error {
	message := kafka.Message{
		Topic: topic,
		Value: rawMessage,
	}
	return k.Writer.WriteMessages(ctx, message)
}

func (k *KafkaDriver) Consume(ctx context.Context, topic, group string, handler queue.ConsumeHandler) error {
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers: k.brokers,
		GroupID: group,
		Topic:   topic,
		// auto commit
		// MaxBytes:       10e6,        // 10MB
		// CommitInterval: time.Second, // flushes commits to Kafka every second
	})

	for {
		m, err := r.FetchMessage(ctx)
		if err != nil {
			fmt.Printf("[kafka.fetch] topic: %s, group: %s, error: %v\n", topic, group, err)
			time.Sleep(time.Second)
			continue
		}

		err = handler(m.Value)
		if err != nil {
			time.Sleep(100 * time.Microsecond)
			continue
		}

		err = r.CommitMessages(ctx, m)
		if err != nil {
			log.Printf("[kafka.commit] topic: %s, group: %s, key: %s, error: %v\n", topic, group, m.Key, err)
			time.Sleep(time.Second)
			continue
		}
	}
}

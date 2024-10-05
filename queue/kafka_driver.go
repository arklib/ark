package queue

import (
	"context"
	"log"
	"strings"

	"github.com/IBM/sarama"
)

type (
	KafkaDriver struct {
		Driver
		brokers  []string
		producer sarama.SyncProducer
	}

	kafkaConsumer struct {
		handler func(rawMessage []byte) error
		ready   chan bool
	}
)

func NewKafkaDriver(brokers []string) *KafkaDriver {
	producer, err := sarama.NewSyncProducer(brokers, nil)
	if err != nil {
		log.Fatalf("Failed to start producer: %v", err)
	}

	return &KafkaDriver{
		producer: producer,
		brokers:  brokers,
	}
}

func (k *KafkaDriver) Produce(ctx context.Context, topic string, rawMessage []byte) error {
	message := &sarama.ProducerMessage{
		Topic: topic,
		Value: sarama.StringEncoder(rawMessage),
	}
	_, _, err := k.producer.SendMessage(message)
	return err
}

func (k *KafkaDriver) Consume(ctx context.Context, topic, group string, handler ConsumeTaskHandler) error {
	config := sarama.NewConfig()
	config.Consumer.Offsets.Initial = sarama.OffsetOldest

	client, err := sarama.NewConsumerGroup(k.brokers, group, config)
	if err != nil {
		return err
	}

	consumer := &kafkaConsumer{
		handler: handler,
		ready:   make(chan bool),
	}
	for {
		err = client.Consume(ctx, strings.Split(topic, ","), consumer)
		if err != nil {
			return err
		}
		<-consumer.ready
	}
}

func (consumer *kafkaConsumer) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for message := range claim.Messages() {
		err := consumer.handler(message.Value)
		if err == nil {
			session.MarkMessage(message, "")
		}
	}
	return nil
}

func (consumer *kafkaConsumer) Setup(sarama.ConsumerGroupSession) error {
	close(consumer.ready)
	return nil
}

func (consumer *kafkaConsumer) Cleanup(sarama.ConsumerGroupSession) error {
	return nil
}

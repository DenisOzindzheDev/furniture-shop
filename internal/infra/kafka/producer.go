package kafka

import (
	"context"
	"encoding/json"
	"log"

	"github.com/segmentio/kafka-go"
)

type EventType string

const (
	EventOrderCreated   EventType = "order.created"
	EventOrderPaid      EventType = "order.paid"
	EventOrderShipped   EventType = "order.shipped"
	EventUserRegistered EventType = "user.registered"
)

type Event struct {
	Type EventType   `json:"type"`
	Data interface{} `json:"data"`
}

type Producer struct {
	writer *kafka.Writer
}

func NewProducer(brokers []string, topic string) *Producer {
	writer := &kafka.Writer{
		Addr:     kafka.TCP(brokers...),
		Topic:    topic,
		Balancer: &kafka.LeastBytes{},
	}

	return &Producer{writer: writer}
}

func (p *Producer) SendEvent(ctx context.Context, eventType EventType, data interface{}) error {
	event := Event{
		Type: eventType,
		Data: data,
	}

	message, err := json.Marshal(event)
	if err != nil {
		return err
	}

	err = p.writer.WriteMessages(ctx, kafka.Message{
		Value: message,
	})

	if err != nil {
		log.Printf("Failed to send kafka event: %v", err)
		return err
	}

	log.Printf("Sent kafka event: %s", eventType)
	return nil
}

func (p *Producer) Close() error {
	return p.writer.Close()
}

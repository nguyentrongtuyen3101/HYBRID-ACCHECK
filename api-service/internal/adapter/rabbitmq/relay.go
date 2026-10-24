// Package rabbitmq đẩy sự kiện từ outbox sang RabbitMQ (topic exchange).
package rabbitmq

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	Exchange = "reqsentinel.events"
	// Sự kiện không route được (chưa service nào bind queue) được giữ ở đây thay vì bị bỏ.
	UnroutedExchange = "reqsentinel.events.unrouted"
	UnroutedQueue    = "reqsentinel.events.unrouted"
)

// ExchangeArgs phải giống hệt ở mọi service khai báo exchange này.
var ExchangeArgs = amqp.Table{"alternate-exchange": UnroutedExchange}

// OutboxSource là nguồn sự kiện chờ gửi (hiện thực: postgres.Outbox).
type OutboxSource interface {
	ProcessPending(ctx context.Context, limit int, send func(id, eventType string, payload []byte) error) (int, error)
	Cleanup(ctx context.Context, olderThan time.Duration) error
}

type Relay struct {
	url      string
	source   OutboxSource
	log      *slog.Logger
	interval time.Duration
}

func NewRelay(url string, source OutboxSource, log *slog.Logger) *Relay {
	return &Relay{url: url, source: source, log: log, interval: time.Second}
}

// Run chạy đến khi ctx bị huỷ; mất kết nối thì tự kết nối lại.
func (r *Relay) Run(ctx context.Context) {
	backoff := time.Second
	for ctx.Err() == nil {
		err := r.session(ctx)
		if ctx.Err() != nil {
			return
		}
		r.log.Warn("rabbitmq relay disconnected, retrying", "err", err, "in", backoff)
		select {
		case <-ctx.Done():
			return
		case <-time.After(backoff):
		}
		if backoff < 30*time.Second {
			backoff *= 2
		}
	}
}

func (r *Relay) session(ctx context.Context) error {
	conn, err := amqp.Dial(r.url)
	if err != nil {
		return err
	}
	defer conn.Close()
	ch, err := conn.Channel()
	if err != nil {
		return err
	}
	defer ch.Close()
	if err := declareTopology(ch); err != nil {
		return err
	}
	if err := ch.Confirm(false); err != nil {
		return err
	}
	r.log.Info("rabbitmq relay connected", "exchange", Exchange)
	closed := conn.NotifyClose(make(chan *amqp.Error, 1))

	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()
	cleanup := time.NewTicker(time.Hour)
	defer cleanup.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case err := <-closed:
			return fmt.Errorf("connection closed: %v", err)
		case <-cleanup.C:
			if err := r.source.Cleanup(ctx, 7*24*time.Hour); err != nil {
				r.log.Warn("outbox cleanup failed", "err", err)
			}
		case <-ticker.C:
			for {
				n, err := r.source.ProcessPending(ctx, 100, func(id, eventType string, payload []byte) error {
					return publish(ctx, ch, id, eventType, payload)
				})
				if err != nil {
					return err
				}
				if n < 100 {
					break
				}
			}
		}
	}
}

func declareTopology(ch *amqp.Channel) error {
	if err := ch.ExchangeDeclare(UnroutedExchange, "fanout", true, false, false, false, nil); err != nil {
		return err
	}
	if _, err := ch.QueueDeclare(UnroutedQueue, true, false, false, false, nil); err != nil {
		return err
	}
	if err := ch.QueueBind(UnroutedQueue, "", UnroutedExchange, false, nil); err != nil {
		return err
	}
	return ch.ExchangeDeclare(Exchange, "topic", true, false, false, false, ExchangeArgs)
}

// publish gửi 1 sự kiện (persistent) và chờ broker xác nhận.
func publish(ctx context.Context, ch *amqp.Channel, id, eventType string, payload []byte) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	conf, err := ch.PublishWithDeferredConfirmWithContext(ctx, Exchange, eventType, false, false, amqp.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp.Persistent,
		MessageId:    id,
		Type:         eventType,
		Timestamp:    time.Now(),
		Body:         payload,
	})
	if err != nil {
		return err
	}
	ok, err := conf.WaitContext(ctx)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("broker nacked event %s", id)
	}
	return nil
}

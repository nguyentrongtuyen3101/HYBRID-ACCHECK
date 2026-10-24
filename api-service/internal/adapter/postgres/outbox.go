package postgres

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5"

	"reqsentinel/api/internal/domain"
)

// Outbox hiện thực app.EventPublisher: sự kiện được ghi cùng transaction với
// thay đổi dữ liệu, relay đẩy sang RabbitMQ sau (transactional outbox pattern).
type Outbox struct{ db *DB }

func NewOutbox(db *DB) *Outbox { return &Outbox{db: db} }

// EventMessage là định dạng JSON gửi sang RabbitMQ (hợp đồng với notification-service).
type EventMessage struct {
	ID         string         `json:"id"`
	Type       string         `json:"type"`
	OccurredAt time.Time      `json:"occurred_at"`
	Actor      *ActorMessage  `json:"actor"`
	Recipients []string       `json:"recipients"`
	Data       map[string]any `json:"data"`
}

type ActorMessage struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
}

func (o *Outbox) Publish(ctx context.Context, events ...domain.Event) error {
	for _, e := range events {
		msg := EventMessage{ID: e.ID, Type: e.Type, OccurredAt: e.OccurredAt, Recipients: e.Recipients, Data: e.Data}
		if e.Actor != nil {
			msg.Actor = &ActorMessage{ID: e.Actor.ID, Email: e.Actor.Email, Name: e.Actor.DisplayName()}
		}
		payload, err := json.Marshal(msg)
		if err != nil {
			return err
		}
		if _, err := o.db.q(ctx).Exec(ctx,
			`INSERT INTO outbox_events (id, event_type, payload, created_at) VALUES ($1, $2, $3, $4)`,
			e.ID, e.Type, payload, e.OccurredAt); err != nil {
			return err
		}
	}
	return nil
}

type pendingEvent struct {
	ID      string
	Type    string
	Payload []byte
}

// ProcessPending khoá 1 lô sự kiện chưa gửi, gọi send cho từng cái và đánh dấu kết quả.
// FOR UPDATE SKIP LOCKED cho phép nhiều instance chạy relay song song.
func (o *Outbox) ProcessPending(ctx context.Context, limit int, send func(id, eventType string, payload []byte) error) (int, error) {
	sent := 0
	err := pgx.BeginFunc(ctx, o.db.Pool, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `
			SELECT id, event_type, payload FROM outbox_events
			WHERE published_at IS NULL AND attempts < 20
			ORDER BY created_at LIMIT $1 FOR UPDATE SKIP LOCKED`, limit)
		if err != nil {
			return err
		}
		events, err := collect(rows, func(s scanner) (pendingEvent, error) {
			var e pendingEvent
			return e, s.Scan(&e.ID, &e.Type, &e.Payload)
		})
		if err != nil {
			return err
		}
		for _, e := range events {
			if err := send(e.ID, e.Type, e.Payload); err != nil {
				if _, uerr := tx.Exec(ctx, `UPDATE outbox_events SET attempts = attempts + 1, last_error = $2 WHERE id = $1`,
					e.ID, err.Error()); uerr != nil {
					return uerr
				}
				// Giữ thứ tự: dừng lô này, lần sau thử lại
				return nil
			}
			if _, err := tx.Exec(ctx, `UPDATE outbox_events SET published_at = now(), attempts = attempts + 1 WHERE id = $1`,
				e.ID); err != nil {
				return err
			}
			sent++
		}
		return nil
	})
	return sent, err
}

// Cleanup xoá sự kiện đã gửi quá hạn giữ lại.
func (o *Outbox) Cleanup(ctx context.Context, olderThan time.Duration) error {
	_, err := o.db.Pool.Exec(ctx, `DELETE FROM outbox_events WHERE published_at < now() - make_interval(secs => $1)`,
		olderThan.Seconds())
	return err
}

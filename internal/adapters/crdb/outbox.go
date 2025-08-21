package crdb

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type OutboxRecord struct {
	ID            uuid.UUID
	AggregateType string
	AggregateID   uuid.UUID
	EventType     string
	Payload       []byte
	CreatedAt     time.Time
	PublishedAt   *time.Time
	Status        string // NEW, PUBLISHED, FAILED
	DedupeKey     string
}

func (r *Repository) InsertOutbox(ctx context.Context, tx pgx.Tx, record OutboxRecord) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO outbox (id, aggregate_type, aggregate_id, event_type, payload_json, status, dedupe_key)
		VALUES ($1, $2, $3, $4, $5, 'NEW', $6)
	`, record.ID, record.AggregateType, record.AggregateID, record.EventType, record.Payload, record.DedupeKey)
	return err
}

func (r *Repository) GetUnpublishedOutbox(ctx context.Context, limit int) ([]OutboxRecord, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, aggregate_type, aggregate_id, event_type, payload_json, created_at, published_at, status, dedupe_key
		FROM outbox WHERE status = 'NEW' ORDER BY created_at ASC LIMIT $1 FOR UPDATE SKIP LOCKED
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []OutboxRecord
	for rows.Next() {
		var rec OutboxRecord
		err := rows.Scan(&rec.ID, &rec.AggregateType, &rec.AggregateID, &rec.EventType, &rec.Payload, &rec.CreatedAt, &rec.PublishedAt, &rec.Status, &rec.DedupeKey)
		if err != nil {
			return nil, err
		}
		records = append(records, rec)
	}
	return records, nil
}

func (r *Repository) MarkPublished(ctx context.Context, id uuid.UUID, publishedAt time.Time, dedupeKey string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE outbox SET status = 'PUBLISHED', published_at = $2, dedupe_key = $3 WHERE id = $1
	`, id, publishedAt, dedupeKey)
	return err
}

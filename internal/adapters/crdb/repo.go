package crdb

import (
	"context"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/robertarktes/ticket-reservations-and-orders/internal/domain"
	"golang.org/x/sync/errgroup"
)

const (
	SerializationFailureCode = "40001"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) WithTx(ctx context.Context, fn func(tx pgx.Tx) error) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, "SET TRANSACTION ISOLATION LEVEL SERIALIZABLE")
	if err != nil {
		return err
	}

	err = fn(tx)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == SerializationFailureCode {
			return domain.ErrSerializationFailure
		}
		return err
	}

	return tx.Commit(ctx)
}

func (r *Repository) CreateHold(ctx context.Context, tx pgx.Tx, hold domain.Hold) error {
	for _, seat := range hold.Seats {
		result, err := tx.Exec(ctx, `
			INSERT INTO holds (id, event_id, seat_no, user_id, expires_at, status)
			VALUES ($1, $2, $3, $4, $5, 'ACTIVE')
			ON CONFLICT (event_id, seat_no) WHERE status = 'ACTIVE' DO NOTHING
			RETURNING id
		`, hold.ID, hold.EventID, seat, hold.UserID, hold.ExpiresAt)
		if err != nil {
			return err
		}
		if result.RowsAffected() == 0 {
			return domain.ErrConflict
		}
	}
	return nil
}

func (r *Repository) CreateOrder(ctx context.Context, tx pgx.Tx, order domain.Order) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO orders (id, user_id, status, total_amount)
		VALUES ($1, $2, 'PENDING', $3)
	`, order.ID, order.UserID, order.TotalAmount)
	if err != nil {
		return err
	}

	g, gctx := errgroup.WithContext(ctx)
	for _, item := range order.Items {
		item := item
		g.Go(func() error {
			_, err := tx.Exec(gctx, `
				INSERT INTO order_items (order_id, event_id, seat_no, price)
				VALUES ($1, $2, $3, $4)
			`, order.ID, item.EventID, item.SeatNo, item.Price)
			return err
		})
	}
	if err := g.Wait(); err != nil {
		return err
	}

	for _, item := range order.Items {
		_, err := tx.Exec(ctx, `
			UPDATE holds SET status = 'RELEASED'
			WHERE event_id = $1 AND seat_no = $2 AND status = 'ACTIVE'
		`, item.EventID, item.SeatNo)
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *Repository) UpdateOrderStatus(ctx context.Context, orderID uuid.UUID, status string) error {
	result, err := r.pool.Exec(ctx, `
		UPDATE orders SET status = $2 WHERE id = $1
	`, orderID, status)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *Repository) GetExpiredHolds(ctx context.Context, now time.Time) ([]domain.Hold, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, event_id, seat_no, user_id, expires_at
		FROM holds WHERE status = 'ACTIVE' AND expires_at <= $1
	`, now)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var holds []domain.Hold
	currentHoldID := uuid.UUID{}
	var currentHold *domain.Hold

	for rows.Next() {
		var id, eventID, userID uuid.UUID
		var seatNo string
		var expiresAt time.Time
		if err := rows.Scan(&id, &eventID, &seatNo, &userID, &expiresAt); err != nil {
			return nil, err
		}
		if id != currentHoldID {
			if currentHold != nil {
				holds = append(holds, *currentHold)
			}
			currentHold = &domain.Hold{
				ID:        id,
				EventID:   eventID,
				UserID:    userID,
				ExpiresAt: expiresAt,
				Seats:     []string{},
			}
			currentHoldID = id
		}
		currentHold.Seats = append(currentHold.Seats, seatNo)
	}
	if currentHold != nil {
		holds = append(holds, *currentHold)
	}
	return holds, nil
}

func (r *Repository) ReleaseHold(ctx context.Context, holdID uuid.UUID) error {
	result, err := r.pool.Exec(ctx, `
		UPDATE holds SET status = 'RELEASED' WHERE id = $1 AND status = 'ACTIVE'
	`, holdID)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *Repository) GetOrder(ctx context.Context, orderID uuid.UUID) (*domain.Order, error) {
	var order domain.Order
	err := r.pool.QueryRow(ctx, `
		SELECT id, user_id, status, total_amount
		FROM orders WHERE id = $1
	`, orderID).Scan(&order.ID, &order.UserID, &order.Status, &order.TotalAmount)
	if err == pgx.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	rows, err := r.pool.Query(ctx, `
		SELECT event_id, seat_no, price
		FROM order_items WHERE order_id = $1
	`, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var item domain.OrderItem
		if err := rows.Scan(&item.EventID, &item.SeatNo, &item.Price); err != nil {
			return nil, err
		}
		order.Items = append(order.Items, item)
	}

	return &order, nil
}

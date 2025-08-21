package crdb_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/robertarktes/ticket-reservations-and-orders/internal/adapters/crdb"
	"github.com/robertarktes/ticket-reservations-and-orders/internal/domain"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

func TestRepository_CreateHold(t *testing.T) {
	ctx := context.Background()

	crdbContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "cockroachdb/cockroach:v24.1.1",
			Cmd:          []string{"start-single-node", "--insecure"},
			ExposedPorts: []string{"26257/tcp"},
			WaitingFor:   wait.ForHTTP("/health?ready=1").WithPort("8080"),
		},
		Started: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer crdbContainer.Terminate(ctx)

	dsn, err := crdbContainer.Endpoint(ctx, "postgresql")
	if err != nil {
		t.Fatal(err)
	}

	pool, err := pgxpool.New(ctx, dsn+"/tro?sslmode=disable")
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	_, err = pool.Exec(ctx, `
		CREATE DATABASE IF NOT EXISTS tro;
		CREATE TABLE IF NOT EXISTS tro.holds (
			id UUID PRIMARY KEY,
			event_id UUID,
			seat_no TEXT,
			user_id UUID,
			expires_at TIMESTAMPTZ,
			status TEXT CHECK (status IN ('ACTIVE', 'EXPIRED', 'RELEASED')),
			UNIQUE (event_id, seat_no) WHERE status = 'ACTIVE'
		);
	`)
	if err != nil {
		t.Fatal(err)
	}

	repo := crdb.NewRepository(pool)

	hold := domain.Hold{
		ID:        uuid.New(),
		EventID:   uuid.New(),
		Seats:     []string{"A1", "A2"},
		UserID:    uuid.New(),
		ExpiresAt: time.Now().Add(5 * time.Minute),
	}

	err = repo.WithTx(ctx, func(tx pgx.Tx) error {
		return repo.CreateHold(ctx, tx, hold)
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	conflictHold := domain.Hold{
		ID:        uuid.New(),
		EventID:   hold.EventID,
		Seats:     []string{"A1"},
		UserID:    uuid.New(),
		ExpiresAt: time.Now().Add(5 * time.Minute),
	}
	err = repo.WithTx(ctx, func(tx pgx.Tx) error {
		return repo.CreateHold(ctx, tx, conflictHold)
	})
	if !errors.Is(err, domain.ErrConflict) {
		t.Errorf("expected conflict error, got %v", err)
	}
}

func TestRepository_CreateOrder(t *testing.T) {
	ctx := context.Background()

	crdbContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "cockroachdb/cockroach:v24.1.1",
			Cmd:          []string{"start-single-node", "--insecure"},
			ExposedPorts: []string{"26257/tcp"},
			WaitingFor:   wait.ForHTTP("/health?ready=1").WithPort("8080"),
		},
		Started: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer crdbContainer.Terminate(ctx)

	dsn, err := crdbContainer.Endpoint(ctx, "postgresql")
	if err != nil {
		t.Fatal(err)
	}

	pool, err := pgxpool.New(ctx, dsn+"/tro?sslmode=disable")
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	_, err = pool.Exec(ctx, `
		CREATE DATABASE IF NOT EXISTS tro;
		CREATE TABLE IF NOT EXISTS tro.orders (
			id UUID PRIMARY KEY,
			user_id UUID,
			status TEXT CHECK (status IN ('PENDING', 'CONFIRMED', 'FAILED')),
			total_amount NUMERIC
		);
		CREATE TABLE IF NOT EXISTS tro.order_items (
			order_id UUID,
			event_id UUID,
			seat_no TEXT,
			price NUMERIC,
			PRIMARY KEY (order_id, event_id, seat_no)
		);
		CREATE TABLE IF NOT EXISTS tro.holds (
			id UUID PRIMARY KEY,
			event_id UUID,
			seat_no TEXT,
			user_id UUID,
			expires_at TIMESTAMPTZ,
			status TEXT CHECK (status IN ('ACTIVE', 'EXPIRED', 'RELEASED')),
			UNIQUE (event_id, seat_no) WHERE status = 'ACTIVE'
		);
	`)
	if err != nil {
		t.Fatal(err)
	}

	repo := crdb.NewRepository(pool)

	eventID := uuid.New()
	order := domain.Order{
		ID:          uuid.New(),
		UserID:      uuid.New(),
		Status:      "PENDING",
		TotalAmount: 200.0,
		Items: []domain.OrderItem{
			{EventID: eventID, SeatNo: "A1", Price: 100.0},
			{EventID: eventID, SeatNo: "A2", Price: 100.0},
		},
	}

	hold := domain.Hold{
		ID:        uuid.New(),
		EventID:   eventID,
		Seats:     []string{"A1", "A2"},
		UserID:    order.UserID,
		ExpiresAt: time.Now().Add(5 * time.Minute),
	}
	err = repo.WithTx(ctx, func(tx pgx.Tx) error {
		return repo.CreateHold(ctx, tx, hold)
	})
	if err != nil {
		t.Fatal(err)
	}

	err = repo.WithTx(ctx, func(tx pgx.Tx) error {
		return repo.CreateOrder(ctx, tx, order)
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	fetched, err := repo.GetOrder(ctx, order.ID)
	if err != nil {
		t.Fatal(err)
	}
	if fetched.Status != "PENDING" || len(fetched.Items) != 2 {
		t.Errorf("expected order with 2 items and PENDING, got %v with %d items", fetched.Status, len(fetched.Items))
	}
}

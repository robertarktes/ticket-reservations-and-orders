package domain

import (
	"time"

	"github.com/google/uuid"
)

type Hold struct {
	ID        uuid.UUID
	EventID   uuid.UUID
	Seats     []string
	UserID    uuid.UUID
	ExpiresAt time.Time
}

type Order struct {
	ID          uuid.UUID
	UserID      uuid.UUID
	Status      string
	TotalAmount float64
	Items       []OrderItem
}

type OrderItem struct {
	EventID uuid.UUID
	SeatNo  string
	Price   float64
}

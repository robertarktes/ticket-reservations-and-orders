package domain

import "github.com/google/uuid"

func NewOrder(eventID uuid.UUID, seats []string, userID uuid.UUID, paymentMethod string) Order {
	items := make([]OrderItem, len(seats))
	for i, seat := range seats {
		items[i] = OrderItem{EventID: eventID, SeatNo: seat, Price: 0.0}
	}
	return Order{
		ID:          uuid.New(),
		UserID:      userID,
		Status:      "PENDING",
		TotalAmount: 0.0,
		Items:       items,
	}
}

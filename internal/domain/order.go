package domain

import "github.com/google/uuid"

func NewOrder(eventID uuid.UUID, seats []string, userID uuid.UUID, paymentMethod string) Order {
	// Calculate total, create items
	items := make([]OrderItem, len(seats))
	for i, seat := range seats {
		items[i] = OrderItem{EventID: eventID, SeatNo: seat, Price: 100.0} // mock price
	}
	return Order{
		ID:          uuid.New(),
		UserID:      userID,
		Status:      "PENDING",
		TotalAmount: float64(len(seats)) * 100.0,
		Items:       items,
	}
}

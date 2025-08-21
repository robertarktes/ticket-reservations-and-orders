package domain

import (
	"time"

	"github.com/google/uuid"
)

func NewHold(eventID uuid.UUID, seats []string, userID uuid.UUID, ttl time.Duration) Hold {
	return Hold{
		ID:        uuid.New(),
		EventID:   eventID,
		Seats:     seats,
		UserID:    userID,
		ExpiresAt: time.Now().Add(ttl),
	}
}

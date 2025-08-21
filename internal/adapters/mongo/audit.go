package mongo

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/robertarktes/ticket-reservations-and-orders/internal/domain"
	"github.com/robertarktes/ticket-reservations-and-orders/internal/observability"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type AuditLogger struct {
	coll   *mongo.Collection
	logger observability.Logger
}

func NewAuditLogger(db *mongo.Database, logger observability.Logger) *AuditLogger {
	return &AuditLogger{
		coll:   db.Collection("audit_logs"),
		logger: logger,
	}
}

type AuditLog struct {
	ID        uuid.UUID `bson:"_id"`
	Action    string    `bson:"action"`
	UserID    uuid.UUID `bson:"user_id"`
	Timestamp time.Time `bson:"timestamp"`
	Data      bson.M    `bson:"data"`
}

func (a *AuditLogger) LogEvent(ctx context.Context, action string, userID uuid.UUID, data map[string]interface{}) error {
	log := AuditLog{
		ID:        uuid.New(),
		Action:    action,
		UserID:    userID,
		Timestamp: time.Now(),
		Data:      bson.M(data),
	}
	_, err := a.coll.InsertOne(ctx, log)
	if err != nil {
		a.logger.Error("failed to insert audit log", err)
		return err
	}
	return nil
}

func (a *AuditLogger) LogHold(ctx context.Context, hold domain.Hold) error {
	data := map[string]interface{}{
		"hold_id":    hold.ID,
		"event_id":   hold.EventID,
		"seats":      hold.Seats,
		"expires_at": hold.ExpiresAt.Format(time.RFC3339),
	}
	return a.LogEvent(ctx, "hold.created", hold.UserID, data)
}

func (a *AuditLogger) LogOrder(ctx context.Context, order domain.Order) error {
	data := map[string]interface{}{
		"order_id": order.ID,
		"status":   order.Status,
		"total":    order.TotalAmount,
		"items":    order.Items,
	}
	return a.LogEvent(ctx, "order.created", order.UserID, data)
}

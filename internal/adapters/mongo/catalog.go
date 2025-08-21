package mongo

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/robertarktes/ticket-reservations-and-orders/internal/observability"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type CatalogRepository struct {
	coll   *mongo.Collection
	logger observability.Logger
}

func NewCatalogRepository(db *mongo.Database, logger observability.Logger) *CatalogRepository {
	return &CatalogRepository{
		coll:   db.Collection("events"),
		logger: logger,
	}
}

type EventDoc struct {
	ID          uuid.UUID `bson:"_id"`
	Name        string    `bson:"name"`
	Description string    `bson:"description"`
	Venue       string    `bson:"venue"`
	Date        time.Time `bson:"date"`
	Seats       []SeatDoc `bson:"seats"`
	CreatedAt   time.Time `bson:"created_at"`
	UpdatedAt   time.Time `bson:"updated_at"`
}

type SeatDoc struct {
	Number    string  `bson:"number"`
	Row       string  `bson:"row"`
	Section   string  `bson:"section"`
	Price     float64 `bson:"price"`
	Available bool    `bson:"available"`
}

func (c *CatalogRepository) GetEvent(ctx context.Context, id uuid.UUID) (*EventDoc, error) {
	var event EventDoc
	err := c.coll.FindOne(ctx, bson.M{"_id": id}).Decode(&event)
	if err != nil {
		c.logger.Error("failed to get event", err)
		return nil, err
	}
	return &event, nil
}

func (c *CatalogRepository) CreateEvent(ctx context.Context, event EventDoc) error {
	event.CreatedAt = time.Now()
	event.UpdatedAt = time.Now()
	_, err := c.coll.InsertOne(ctx, event)
	if err != nil {
		c.logger.Error("failed to create event", err)
		return err
	}
	return nil
}

func (c *CatalogRepository) UpdateEventAvailability(ctx context.Context, id uuid.UUID, available bool) error {
	_, err := c.coll.UpdateOne(
		ctx,
		bson.M{"_id": id},
		bson.M{"$set": bson.M{"available": available, "updated_at": time.Now()}},
	)
	if err != nil {
		c.logger.Error("failed to update event availability", err)
		return err
	}
	return nil
}

package http

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/robertarktes/ticket-reservations-and-orders/internal/adapters/crdb"
	"github.com/robertarktes/ticket-reservations-and-orders/internal/adapters/mongo"
	"github.com/robertarktes/ticket-reservations-and-orders/internal/adapters/rabbit"
	redisadapter "github.com/robertarktes/ticket-reservations-and-orders/internal/adapters/redis"
	"github.com/robertarktes/ticket-reservations-and-orders/internal/config"
	"github.com/robertarktes/ticket-reservations-and-orders/internal/domain"
	"github.com/robertarktes/ticket-reservations-and-orders/internal/idempotency"
)

type Handlers struct {
	cfg          *config.Config
	repo         *crdb.Repository
	redis        *redisadapter.Cache
	idemp        *idempotency.Idempotency
	mongoCatalog *mongo.CatalogRepository
	rabbitPub    *rabbit.Publisher
}

func NewHandlers(cfg *config.Config, repo *crdb.Repository, redis *redisadapter.Cache, idemp *idempotency.Idempotency, mongoCatalog *mongo.CatalogRepository, rabbitPub *rabbit.Publisher) *Handlers {
	return &Handlers{
		cfg:          cfg,
		repo:         repo,
		redis:        redis,
		idemp:        idemp,
		mongoCatalog: mongoCatalog,
		rabbitPub:    rabbitPub,
	}
}

func (h *Handlers) CreateHold(w http.ResponseWriter, r *http.Request) {
	key := r.Header.Get("Idempotency-Key")
	existing, err := h.idemp.Get(r.Context(), key)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if existing != nil {
		w.WriteHeader(existing.Status)
		w.Write(existing.Result)
		return
	}

	var req struct {
		EventID uuid.UUID `json:"event_id"`
		Seats   []string  `json:"seats"`
		UserID  uuid.UUID `json:"user_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	_, err = h.mongoCatalog.GetEvent(r.Context(), req.EventID)
	if err != nil {
		http.Error(w, "event not found", http.StatusNotFound)
		return
	}

	hold := domain.NewHold(req.EventID, req.Seats, req.UserID, h.cfg.HoldTTL)

	err = h.repo.WithTx(r.Context(), func(tx pgx.Tx) error {
		for _, seat := range hold.Seats {
			ok, err := h.redis.SetHoldLock(r.Context(), hold.EventID.String(), seat, hold.UserID.String(), h.cfg.HoldTTL)
			if err != nil {
				return err
			}
			if !ok {
				return domain.ErrConflict
			}
		}
		return h.repo.CreateHold(r.Context(), tx, hold)
	})
	if errors.Is(err, domain.ErrSerializationFailure) {
		http.Error(w, "conflict, try again", http.StatusConflict)
		return
	}
	if errors.Is(err, domain.ErrConflict) {
		http.Error(w, "seats already held", http.StatusConflict)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resp := map[string]interface{}{
		"hold_id":    hold.ID,
		"expires_at": hold.ExpiresAt.Format(time.RFC3339),
	}
	data, _ := json.Marshal(resp)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	w.Write(data)

	h.idemp.Set(r.Context(), key, idempotency.Response{Status: http.StatusCreated, Result: data})
}

func (h *Handlers) CreateOrder(w http.ResponseWriter, r *http.Request) {
	key := r.Header.Get("Idempotency-Key")
	existing, err := h.idemp.Get(r.Context(), key)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if existing != nil {
		w.WriteHeader(existing.Status)
		w.Write(existing.Result)
		return
	}

	var req struct {
		EventID       uuid.UUID `json:"event_id"`
		Seats         []string  `json:"seats"`
		UserID        uuid.UUID `json:"user_id"`
		PaymentMethod string    `json:"payment_method"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	order := domain.NewOrder(req.EventID, req.Seats, req.UserID, req.PaymentMethod)

	err = h.repo.WithTx(r.Context(), func(tx pgx.Tx) error {
		if err := h.repo.CreateOrder(r.Context(), tx, order); err != nil {
			return err
		}
		payload, _ := json.Marshal(map[string]interface{}{"order_id": order.ID})
		outboxRec := crdb.OutboxRecord{
			ID:            uuid.New(),
			AggregateType: "order",
			AggregateID:   order.ID,
			EventType:     "order.created",
			Payload:       payload,
			DedupeKey:     uuid.New().String(),
		}
		return h.repo.InsertOutbox(r.Context(), tx, outboxRec)
	})
	if err != nil {
		if errors.Is(err, domain.ErrConflict) {
			http.Error(w, "conflict", http.StatusConflict)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resp := map[string]interface{}{
		"order_id": order.ID,
		"status":   order.Status,
	}
	data, _ := json.Marshal(resp)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	w.Write(data)

	h.idemp.Set(r.Context(), key, idempotency.Response{Status: http.StatusAccepted, Result: data})
}

func (h *Handlers) GetOrder(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	order, err := h.repo.GetOrder(r.Context(), id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			http.Error(w, "order not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resp := map[string]interface{}{
		"status": order.Status,
		"items":  order.Items,
		"total":  order.TotalAmount,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *Handlers) PaymentCallback(w http.ResponseWriter, r *http.Request) {
	var req struct {
		OrderID       uuid.UUID `json:"order_id"`
		Status        string    `json:"status"`
		TransactionID string    `json:"transaction_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	newStatus := "FAILED"
	if req.Status == "SUCCEEDED" {
		newStatus = "CONFIRMED"
	}
	err := h.repo.UpdateOrderStatus(r.Context(), req.OrderID, newStatus)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	eventType := "order." + strings.ToLower(newStatus)
	payload, _ := json.Marshal(map[string]interface{}{"order_id": req.OrderID, "status": newStatus})
	msg := amqp.Publishing{
		MessageId:   uuid.New().String(),
		ContentType: "application/json",
		Body:        payload,
	}
	h.rabbitPub.Publish(r.Context(), eventType, msg)

	w.WriteHeader(http.StatusOK)
}

func (h *Handlers) Healthz(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func (h *Handlers) Readyz(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Ready"))
}

func (h *Handlers) Metrics(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Metrics endpoint - implement Prometheus handler"))
}

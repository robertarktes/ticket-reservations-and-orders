package http

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/robertarktes/ticket-reservations-and-orders/internal/idempotency"
	"github.com/robertarktes/ticket-reservations-and-orders/internal/observability"
	"github.com/robertarktes/ticket-reservations-and-orders/internal/rateLimit"
)

func SetupRouter(h *Handlers, logger observability.Logger, rl *rateLimit.RateLimiter, idemp *idempotency.Idempotency) *chi.Mux {
	r := chi.NewRouter()

	r.Use(middleware.Recoverer)
	r.Use(RequestIDMiddleware)
	r.Use(LoggerMiddleware(logger))
	r.Use(TracingMiddleware)
	r.Use(JWTMiddleware)
	r.Use(RateLimitMiddleware(rl))
	r.Use(IdempotencyMiddleware(idemp))

	r.Post("/v1/holds", h.CreateHold)
	r.Post("/v1/orders", h.CreateOrder)
	r.Get("/v1/orders/{id}", h.GetOrder)
	r.Post("/v1/payments/callback", h.PaymentCallback)
	r.Get("/v1/healthz", h.Healthz)
	r.Get("/v1/readyz", h.Readyz)
	r.Get("/metrics", promhttp.Handler().ServeHTTP)

	return r
}

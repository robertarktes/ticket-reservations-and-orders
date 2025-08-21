package http

import (
	"context"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/robertarktes/ticket-reservations-and-orders/internal/idempotency"
	"github.com/robertarktes/ticket-reservations-and-orders/internal/observability"
	"github.com/robertarktes/ticket-reservations-and-orders/internal/rateLimit"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	otelhttp "go.opentelemetry.io/otel/propagation"
)

func RequestIDMiddleware(next http.Handler) http.Handler {
	return middleware.RequestID(next)
}

func LoggerMiddleware(logger observability.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			reqID := middleware.GetReqID(r.Context())
			entry := logger.WithField("request_id", reqID)
			ctx := context.WithValue(r.Context(), "logger", entry)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func JWTMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r)
	})
}

func IdempotencyMiddleware(idemp *idempotency.Idempotency) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != "POST" {
				next.ServeHTTP(w, r)
				return
			}
			key := r.Header.Get("Idempotency-Key")
			if key == "" {
				http.Error(w, "missing Idempotency-Key", http.StatusBadRequest)
				return
			}
			if len(key) < 16 {
				http.Error(w, "invalid Idempotency-Key", http.StatusBadRequest)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func RateLimitMiddleware(rl *rateLimit.RateLimiter) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID := ""
			ip := r.RemoteAddr
			if !rl.Allow(r.Context(), "user:"+userID, 10, time.Minute) || !rl.Allow(r.Context(), "ip:"+ip, 100, time.Minute) {
				http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func TracingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := otel.GetTextMapPropagator().Extract(r.Context(), otelhttp.HeaderCarrier(r.Header))
		tracer := otel.Tracer("http")
		ctx, span := tracer.Start(ctx, r.Method+" "+r.URL.Path)
		defer span.End()

		span.SetAttributes(
			attribute.String("http.method", r.Method),
			attribute.String("http.url", r.URL.String()),
		)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

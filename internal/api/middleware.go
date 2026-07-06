package api

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"
	"time"

	"go-backend/internal/logging"
	"go-backend/internal/metrics"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

type contextKey string

const (
	UserIDKey  contextKey = "userID"
	IsAdminKey contextKey = "isAdmin"
)

func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := uuid.New().String()
		ctx := context.WithValue(r.Context(), logging.RequestIDKey, id)
		w.Header().Add("X-Request-ID", id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// Recover catches panics so one bad handler cannot take down the process.
func Recover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				slog.ErrorContext(r.Context(), "panic recovered",
					"panic", rec,
					"path", r.URL.Path,
					"stack", string(debug.Stack()),
				)
				respondError(w, http.StatusInternalServerError, "internal server error")
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// RequestLog logs method, path, status, and duration for every request.
func RequestLog(m *metrics.Metrics, next http.Handler) http.Handler {
	tracer := otel.Tracer("go-backend")

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := otel.GetTextMapPropagator().Extract(r.Context(), propagation.HeaderCarrier(r.Header))

		ctx, span := tracer.Start(ctx, fmt.Sprintf("%s %s", r.Method, r.URL.Path),
			trace.WithSpanKind(trace.SpanKindServer),
		)
		defer span.End()

		r = r.WithContext(ctx)

		if m != nil {
			m.RecordActiveRequestStart(ctx)
			defer m.RecordActiveRequestEnd(ctx)
		}

		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)

		duration := time.Since(start)

		route := r.Pattern
		if route == "" {
			route = r.URL.Path
		}

		span.SetAttributes(
			attribute.String("http.method", r.Method),
			attribute.String("http.route", route),
			attribute.Int("http.status_code", rec.status),
		)
		if rec.status >= 500 {
			span.SetStatus(codes.Error, fmt.Sprintf("HTTP %d", rec.status))
		}

		slog.InfoContext(ctx, "request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", rec.status,
			"duration", duration,
		)
		if m != nil {
			m.RecordHttpRequest(ctx, r.Method, route, rec.status, duration, int64(rec.bytesWritten))
		}
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status       int
	bytesWritten int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

func (r *statusRecorder) Write(b []byte) (int, error) {
	n, err := r.ResponseWriter.Write(b)
	r.bytesWritten += n
	return n, err
}

func RequireAuth(jwtSecret []byte, publicRoutes map[string]struct{}) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			route := r.Method + " " + r.URL.Path
			if _, ok := publicRoutes[route]; ok {
				next.ServeHTTP(w, r)
				return
			}

			cookie, err := r.Cookie("token")
			if err != nil {
				respondError(w, http.StatusUnauthorized, "missing auth token")
				return
			}

			claims, err := parseToken(cookie.Value, jwtSecret)
			if err != nil {
				respondError(w, http.StatusUnauthorized, "invalid or expired token")
				return
			}

			sub, ok := claims["sub"].(float64)
			if !ok {
				respondError(w, http.StatusUnauthorized, "invalid token subject")
				return
			}

			isAdmin, _ := claims["is_admin"].(bool)

			ctx := context.WithValue(r.Context(), UserIDKey, int64(sub))
			ctx = context.WithValue(ctx, IsAdminKey, isAdmin)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		isAdmin, _ := r.Context().Value(IsAdminKey).(bool)
		if !isAdmin {
			respondError(w, http.StatusForbidden, "admin access required")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func parseToken(tokenString string, secret []byte) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenString, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return secret, nil
	})
	if err != nil || !token.Valid {
		return nil, err
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, jwt.ErrTokenInvalidClaims
	}
	return claims, nil
}

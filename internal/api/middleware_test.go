package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var uuidRe = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

func TestRequestID(t *testing.T) {
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	RequestID(next).ServeHTTP(w, r)

	if !called {
		t.Error("next handler was not called")
	}

	id := w.Header().Get("X-Request-ID")
	if id == "" {
		t.Error("X-Request-ID header is missing")
	}
	if !uuidRe.MatchString(id) {
		t.Errorf("X-Request-ID %q is not a valid UUID", id)
	}
}

func TestRequireAuth(t *testing.T) {
	secret := []byte("test-secret")

	makeToken := func(claims jwt.MapClaims) string {
		tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		s, err := tok.SignedString(secret)
		if err != nil {
			t.Fatalf("failed to sign token: %v", err)
		}
		return s
	}

	tests := []struct {
		name         string
		publicRoutes map[string]struct{}
		setupRequest func(*http.Request) *http.Request
		wantStatus   int
		wantCalled   bool
		checkCtx     func(*testing.T, context.Context)
	}{
		{
			name:         "public route passes through",
			publicRoutes: map[string]struct{}{"GET /health": {}},
			setupRequest: func(r *http.Request) *http.Request { return r },
			wantStatus:   http.StatusOK,
			wantCalled:   true,
		},
		{
			name:         "missing token returns 401",
			publicRoutes: map[string]struct{}{},
			setupRequest: func(r *http.Request) *http.Request { return r },
			wantStatus:   http.StatusUnauthorized,
			wantCalled:   false,
		},
		{
			name:         "valid token sets context values",
			publicRoutes: map[string]struct{}{},
			setupRequest: func(r *http.Request) *http.Request {
				tok := makeToken(jwt.MapClaims{
					"sub":      float64(1),
					"is_admin": true,
					"exp":      time.Now().Add(time.Hour).Unix(),
				})
				r.AddCookie(&http.Cookie{Name: "token", Value: tok})
				return r
			},
			wantStatus: http.StatusOK,
			wantCalled: true,
			checkCtx: func(t *testing.T, ctx context.Context) {
				t.Helper()
				uid, ok := ctx.Value(UserIDKey).(int64)
				if !ok || uid != 1 {
					t.Errorf("UserIDKey: got %v, want int64(1)", ctx.Value(UserIDKey))
				}
				isAdmin, ok := ctx.Value(IsAdminKey).(bool)
				if !ok || !isAdmin {
					t.Errorf("IsAdminKey: got %v, want true", ctx.Value(IsAdminKey))
				}
			},
		},
		{
			name:         "invalid token returns 401",
			publicRoutes: map[string]struct{}{},
			setupRequest: func(r *http.Request) *http.Request {
				r.AddCookie(&http.Cookie{Name: "token", Value: "invalid-token-string"})
				return r
			},
			wantStatus: http.StatusUnauthorized,
			wantCalled: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var capturedCtx context.Context
			called := false

			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				called = true
				capturedCtx = r.Context()
				w.WriteHeader(http.StatusOK)
			})

			mw := RequireAuth(secret, tc.publicRoutes)

			r := httptest.NewRequest(http.MethodGet, "/health", nil)
			r = tc.setupRequest(r)
			w := httptest.NewRecorder()

			mw(next).ServeHTTP(w, r)

			if w.Code != tc.wantStatus {
				t.Errorf("got status %d, want %d", w.Code, tc.wantStatus)
			}
			if called != tc.wantCalled {
				t.Errorf("next called: got %v, want %v", called, tc.wantCalled)
			}
			if tc.checkCtx != nil && capturedCtx != nil {
				tc.checkCtx(t, capturedCtx)
			}
		})
	}
}

func TestRequireAdmin(t *testing.T) {
	tests := []struct {
		name       string
		isAdmin    bool
		wantStatus int
		wantCalled bool
	}{
		{"admin passes", true, http.StatusOK, true},
		{"non-admin 403", false, http.StatusForbidden, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			called := false
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				called = true
				w.WriteHeader(http.StatusOK)
			})

			r := httptest.NewRequest(http.MethodGet, "/admin", nil)
			ctx := context.WithValue(r.Context(), IsAdminKey, tc.isAdmin)
			r = r.WithContext(ctx)
			w := httptest.NewRecorder()

			RequireAdmin(next).ServeHTTP(w, r)

			if w.Code != tc.wantStatus {
				t.Errorf("got status %d, want %d", w.Code, tc.wantStatus)
			}
			if called != tc.wantCalled {
				t.Errorf("next called: got %v, want %v", called, tc.wantCalled)
			}
		})
	}
}

func TestRecover(t *testing.T) {
	panicking := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("something went wrong")
	})

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	Recover(panicking).ServeHTTP(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("got status %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

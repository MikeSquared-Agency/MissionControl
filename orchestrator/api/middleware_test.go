package api

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestCORSMiddleware_AllowedOrigin(t *testing.T) {
	handler := CORSMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/api/status", nil)
	req.Header.Set("Origin", "https://darlington.dev")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "https://darlington.dev" {
		t.Errorf("expected origin https://darlington.dev, got %q", got)
	}
}

func TestCORSMiddleware_DisallowedOrigin(t *testing.T) {
	handler := CORSMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/api/status", nil)
	req.Header.Set("Origin", "https://evil.com")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("expected no CORS header, got %q", got)
	}
}

func TestCORSMiddleware_Preflight(t *testing.T) {
	handler := CORSMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called for OPTIONS")
	}))

	req := httptest.NewRequest("OPTIONS", "/api/status", nil)
	req.Header.Set("Origin", "https://darlington.dev")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", rec.Code)
	}
}

func TestAuthMiddleware_NoTokenConfigured(t *testing.T) {
	os.Unsetenv("MC_API_TOKEN")

	handler := AuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/api/status", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 when no token configured, got %d", rec.Code)
	}
}

func TestAuthMiddleware_ValidBearerToken(t *testing.T) {
	os.Setenv("MC_API_TOKEN", "test-secret")
	defer os.Unsetenv("MC_API_TOKEN")

	handler := AuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/api/status", nil)
	req.Header.Set("Authorization", "Bearer test-secret")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 with valid token, got %d", rec.Code)
	}
}

func TestAuthMiddleware_InvalidToken(t *testing.T) {
	os.Setenv("MC_API_TOKEN", "test-secret")
	defer os.Unsetenv("MC_API_TOKEN")

	handler := AuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called with invalid token")
	}))

	req := httptest.NewRequest("GET", "/api/status", nil)
	req.Header.Set("Authorization", "Bearer wrong-token")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestAuthMiddleware_QueryParamToken(t *testing.T) {
	os.Setenv("MC_API_TOKEN", "test-secret")
	defer os.Unsetenv("MC_API_TOKEN")

	handler := AuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/api/status?token=test-secret", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 with query param token, got %d", rec.Code)
	}
}

func TestChain(t *testing.T) {
	called := false
	handler := Chain(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = true
			w.WriteHeader(http.StatusOK)
		}),
		CORSMiddleware,
	)

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Origin", "https://darlington.dev")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if !called {
		t.Error("handler was not called")
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "https://darlington.dev" {
		t.Errorf("CORS not applied through chain, got %q", got)
	}
}

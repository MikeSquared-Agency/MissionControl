package api

import (
	"net/http"
	"os"
	"strings"
)

// AllowedOrigins for CORS
var AllowedOrigins = []string{
	"https://darlington.dev",
	"https://www.darlington.dev",
	"http://localhost:3000",
}

// CORSMiddleware adds CORS headers for darlington.dev and local dev
func CORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		for _, allowed := range AllowedOrigins {
			if origin == allowed {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				break
			}
		}
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Credentials", "true")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// AuthMiddleware checks bearer token against MC_API_TOKEN env var.
// If MC_API_TOKEN is not set, auth is disabled (local dev mode).
func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := os.Getenv("MC_API_TOKEN")
		if token == "" {
			// No token configured — auth disabled
			next.ServeHTTP(w, r)
			return
		}

		// Check Authorization header
		auth := r.Header.Get("Authorization")
		if strings.HasPrefix(auth, "Bearer ") {
			if strings.TrimPrefix(auth, "Bearer ") == token {
				next.ServeHTTP(w, r)
				return
			}
		}

		// Check query param fallback
		if r.URL.Query().Get("token") == token {
			next.ServeHTTP(w, r)
			return
		}

		http.Error(w, "Unauthorized", http.StatusUnauthorized)
	})
}

// Chain applies middlewares in order
func Chain(handler http.Handler, middlewares ...func(http.Handler) http.Handler) http.Handler {
	for i := len(middlewares) - 1; i >= 0; i-- {
		handler = middlewares[i](handler)
	}
	return handler
}

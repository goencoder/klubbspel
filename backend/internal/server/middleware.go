package server

import (
	"context"
	"net/http"
)

// Actor middleware: read X-Actor for auditing
const CtxActorKey = ctxKey("actor")

type ctxKey string

func withActor(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), CtxActorKey, r.Header.Get("X-Actor"))
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func ActorFrom(ctx context.Context) string {
	v, _ := ctx.Value(CtxActorKey).(string)
	if v == "" {
		return "anonymous"
	}
	return v
}

// allowCORS configures CORS for development allowing frontend from GitHub
func allowCORS(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set CORS headers for all requests
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "*")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Access-Control-Max-Age", "3600")

		// Handle preflight OPTIONS request
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		// Filter out problematic headers that can cause gRPC-Gateway issues
		filteredHeaders := []string{
			"Connection",
			"Keep-Alive",
			"Proxy-Connection",
			"Transfer-Encoding",
			"Upgrade",
		}

		for _, header := range filteredHeaders {
			if r.Header.Get(header) != "" {
				r.Header.Del(header)
			}
		}

		// Continue with the actual request
		handler.ServeHTTP(w, r)
	})
}

package handler

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
)

type defaultHandler struct {
	log *slog.Logger
}

func NewDefaultHandler(log *slog.Logger) defaultHandler {
	return defaultHandler{
		log: log,
	}
}

func DefaultRoutes() http.Handler {
	r := chi.NewRouter()

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Header().Add("Content-Type", "text/plain")
		w.Write([]byte("ok"))
	})

	r.Get("/resource1", instrumentTotalRequests(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Header().Add("Content-Type", "text/plain")
		w.Write([]byte("resource1\n"))
	}))

	r.Get("/resource2", instrumentTotalRequests(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Header().Add("Content-Type", "text/plain")
		w.Write([]byte("resource2\n"))
	}))

	return r
}

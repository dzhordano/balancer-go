package handler

import (
	"log/slog"
	"math/rand"
	"net/http"
	"time"

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

		// Псевдо-задержка.
		n := rand.Intn(1001)
		if n == 1000 {
			time.Sleep(350 * time.Millisecond)
		} else {
			time.Sleep(time.Duration(rand.Intn(10)) * time.Millisecond)
		}

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

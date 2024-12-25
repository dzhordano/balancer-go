package routes

import (
	"math/rand"
	"net/http"
	"time"

	"github.com/dzhordano/balancer-go/pkg/metrics"
	"github.com/go-chi/chi/v5"
)

func DefaultRoutes() http.Handler {
	r := chi.NewRouter()
	r.Use(metrics.InstrumentConcretePathRequests)

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Header().Add("Content-Type", "text/plain")
		w.Write([]byte("ok"))
	})

	r.Get("/resource1", func(w http.ResponseWriter, r *http.Request) {

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
	})

	r.Get("/resource2", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Header().Add("Content-Type", "text/plain")
		w.Write([]byte("resource2\n"))
	})

	return r
}

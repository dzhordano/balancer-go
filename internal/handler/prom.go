package handler

import (
	"net/http"

	"github.com/dzhordano/balancer-go/internal/config"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	requestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "endpoint"},
	)

	algorithmRequests = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "balancer_algorithm_requests_total",
			Help: "Total requests per balancing algorithm",
		},
		[]string{"algorithm"},
	)
)

func init() {
}

func init() {
	prometheus.MustRegister(requestsTotal)
	prometheus.MustRegister(algorithmRequests)
}

func instrumentHandler(endpoint string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		requestsTotal.WithLabelValues(r.Method, endpoint).Inc()
		algorithmRequests.WithLabelValues(config.NewConfig().BalancingAlg).Inc() // FIXME плохо инициализирую КФГ
		next(w, r)
	}
}

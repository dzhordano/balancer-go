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
	activeRequests = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "http_active_requests",
			Help: "Number of active HTTP requests",
		},
		[]string{"server"},
	)
)

func init() {
	prometheus.MustRegister(requestsTotal)
	prometheus.MustRegister(algorithmRequests)
	prometheus.MustRegister(activeRequests)
}

func instrumentHandler(endpoint string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		requestsTotal.WithLabelValues(r.Method, endpoint).Inc()
		algorithmRequests.WithLabelValues(config.NewConfig("configs/config.yaml").BalancingAlg).Inc() // FIXME плохо инициализирую КФГ
		next(w, r)
	}
}

func instrumentTotalRequests(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		activeRequests.WithLabelValues(r.Host + r.URL.Path).Inc()
		next(w, r)
	}
}

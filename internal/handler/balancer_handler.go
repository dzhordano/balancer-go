package handler

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
)

const (
	roundRobinAlg         = "round_robin"
	weightedRoundRobinAlg = "weighted_round_robin"
	leastConnAlg          = "least_connections"
	hashAlg               = "hash"
	randomAlg             = "random"
)

type Server struct {
	URL    string
	Weight int
}

func NewServer(url string, weight int) *Server {
	return &Server{
		URL:    url,
		Weight: weight,
	}
}

type balancerHandler struct {
	log      *slog.Logger
	balancer Balancer
}

func NewBalancerHandler(log *slog.Logger, servers []Server, alg string, interval, timeout time.Duration) *balancerHandler {
	var balancer Balancer
	switch alg {
	case roundRobinAlg:
		balancer = &RoundRobinBalancer{}
	case weightedRoundRobinAlg:
		balancer = &WeightedRoundRobinBalancer{}
	// case leastConnAlg:
	// 	balancer = &LeastConnectionsBalancer{}
	// case hashAlg:
	// 	balancer = &HashBalancer{}
	// TODO
	// case randomAlg:
	// 	balancer = NewRandomBalancer(servers)
	default:
		log.Error("unknown balancing algorithm", slog.String("algorithm", alg))
		return nil
	}

	balancer.SetServers(servers)

	go balancer.HealthCheck(interval, timeout)

	return &balancerHandler{
		log:      log,
		balancer: balancer,
	}
}

func (h *balancerHandler) Routes() http.Handler {
	r := chi.NewRouter()

	r.Get("/resource1", instrumentHandler("/resource1", h.forwardRequest))
	r.Get("/resource2", instrumentHandler("/resource2", h.forwardRequest))

	return r
}

func (b *balancerHandler) forwardRequest(w http.ResponseWriter, r *http.Request) {
	key := r.RemoteAddr // Используем IP клиента как ключ
	server := b.balancer.SelectServer(key)
	if server == nil {
		http.Error(w, "no available servers", http.StatusServiceUnavailable)
		return
	}

	targetURL := fmt.Sprintf("http://%s%s", server.URL, r.URL.Path)
	fmt.Println("forwarding request to", targetURL)
	req, err := http.NewRequest(r.Method, targetURL, r.Body)
	if err != nil {
		http.Error(w, "failed to create request", http.StatusInternalServerError)
		return
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		http.Error(w, "failed to forward request", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

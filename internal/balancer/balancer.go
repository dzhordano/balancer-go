package balancer

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/dzhordano/balancer-go/internal/server"
	"github.com/dzhordano/balancer-go/pkg/metrics"
	"github.com/go-chi/chi/v5"
)

const (
	roundRobinAlg         = "round_robin"
	weightedRoundRobinAlg = "weighted_round_robin"
	leastConnAlg          = "least_connections"
	hashAlg               = "hash"
	randomAlg             = "random"
)

type Balancer interface {
	SetServers(servers []server.Server)
	SelectServer(args ...interface{}) *server.Server
	DownServers() []server.Server
	AliveServers() []server.Server
	AddAliveServer(server server.Server)
	AddDownServer(server server.Server)
	RemoveDownServer(index int)
	RemoveAliveServer(index int)
}
type balancerHandler struct {
	log      *slog.Logger
	balancer Balancer
}

func NewBalancerHandler(log *slog.Logger, servers []server.Server, alg string) *balancerHandler {
	var balancer Balancer
	switch alg {
	case roundRobinAlg:
		balancer = &RoundRobinBalancer{}
	case weightedRoundRobinAlg:
		balancer = &WeightedRoundRobinBalancer{}
	case leastConnAlg:
		balancer = &LeastConnectionsBalancer{}
	case hashAlg:
		balancer = &HashBalancer{}
	case randomAlg:
		balancer = &RandomBalancer{}
	default:
		log.Error("unknown balancing algorithm", slog.String("algorithm", alg))
		return nil
	}

	balancer.SetServers(servers)

	return &balancerHandler{
		log:      log,
		balancer: balancer,
	}
}

func (h *balancerHandler) Routes() http.Handler {
	r := chi.NewRouter()

	r.Get("/resource1", metrics.InstrumentHandler("/resource1", h.forwardRequest))
	r.Get("/resource2", metrics.InstrumentHandler("/resource2", h.forwardRequest))

	return r
}
func (h *balancerHandler) Balancer() Balancer {
	return h.balancer
}

func (b *balancerHandler) forwardRequest(w http.ResponseWriter, r *http.Request) {
	key := r.RemoteAddr // Используем IP клиента как ключ
	server := b.balancer.SelectServer(key)
	if server == nil {
		http.Error(w, "no available servers", http.StatusServiceUnavailable)
		return
	}

	server.IncrementConnections()
	defer server.DecrementConnections()

	targetURL := fmt.Sprintf("http://%s%s", server.URL, r.URL.Path)
	b.log.Debug("forwarding request to", slog.String("url", targetURL))
	req, err := http.NewRequest(r.Method, targetURL, r.Body)
	if err != nil {
		b.log.Error("failed to create request", slog.String("url", targetURL), slog.String("error", err.Error()))

		http.Error(w, "failed to create request", http.StatusInternalServerError)
		return
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		b.log.Error("failed to forward request", slog.String("url", targetURL), slog.String("error", err.Error()))

		http.Error(w, "failed to forward request", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

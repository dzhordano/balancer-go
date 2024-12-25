package balancer

import (
	"math/rand"
	"sync"

	"github.com/dzhordano/balancer-go/internal/server"
)

type RandomBalancer struct {
	downServers  []server.Server
	aliveServers []server.Server
	mu           sync.Mutex
}

func (rb *RandomBalancer) SetServers(servers []server.Server) {
	rb.mu.Lock()
	defer rb.mu.Unlock()
	rb.aliveServers = servers
}

func (rb *RandomBalancer) SelectServer(args ...interface{}) *server.Server {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	if len(rb.aliveServers) == 0 {
		return nil
	}

	return &rb.aliveServers[rand.Intn(len(rb.aliveServers))]
}

func (rb *RandomBalancer) DownServers() []server.Server {
	rb.mu.Lock()
	defer rb.mu.Unlock()
	return rb.downServers
}

func (rb *RandomBalancer) AliveServers() []server.Server {
	rb.mu.Lock()
	defer rb.mu.Unlock()
	return rb.aliveServers
}

func (rb *RandomBalancer) RemoveAliveServer(index int) {
	rb.mu.Lock()
	defer rb.mu.Unlock()
	rb.aliveServers = append(rb.aliveServers[:index], rb.aliveServers[index+1:]...)
}

func (rb *RandomBalancer) RemoveDownServer(index int) {
	rb.mu.Lock()
	defer rb.mu.Unlock()
	rb.downServers = append(rb.downServers[:index], rb.downServers[index+1:]...)
}

func (rb *RandomBalancer) AddAliveServer(server server.Server) {
	rb.mu.Lock()
	defer rb.mu.Unlock()
	rb.aliveServers = append(rb.aliveServers, server)
}

func (rb *RandomBalancer) AddDownServer(server server.Server) {
	rb.mu.Lock()
	defer rb.mu.Unlock()
	rb.downServers = append(rb.downServers, server)
}

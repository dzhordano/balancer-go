package balancer

import (
	"sync"

	"github.com/dzhordano/balancer-go/internal/server"
)

type RoundRobinBalancer struct {
	downServers  []server.Server
	aliveServers []server.Server
	index        int
	mu           sync.Mutex
}

func (rr *RoundRobinBalancer) SetServers(servers []server.Server) {
	rr.aliveServers = servers
	rr.index = 0
}

func (rr *RoundRobinBalancer) SelectServer(args ...interface{}) *server.Server {
	rr.mu.Lock()
	defer rr.mu.Unlock()

	if len(rr.aliveServers) == 0 {
		return nil
	}

	server := &rr.aliveServers[rr.index]
	rr.index = (rr.index + 1) % len(rr.aliveServers)

	return server
}

func (rr *RoundRobinBalancer) DownServers() []server.Server {
	rr.mu.Lock()
	defer rr.mu.Unlock()
	return rr.downServers
}

func (rr *RoundRobinBalancer) RemoveAliveServer(index int) {
	rr.mu.Lock()
	defer rr.mu.Unlock()
	rr.aliveServers = append(rr.aliveServers[:index], rr.aliveServers[index+1:]...)
	if len(rr.aliveServers) == 0 {
		rr.index = 0
		return
	}

	rr.index = (rr.index) % len(rr.aliveServers)
}

func (rr *RoundRobinBalancer) RemoveDownServer(index int) {
	rr.mu.Lock()
	defer rr.mu.Unlock()
	rr.downServers = append(rr.downServers[:index], rr.downServers[index+1:]...)
}

func (rr *RoundRobinBalancer) AddAliveServer(server server.Server) {
	rr.mu.Lock()
	defer rr.mu.Unlock()
	rr.aliveServers = append(rr.aliveServers, server)
}

func (rr *RoundRobinBalancer) AddDownServer(server server.Server) {
	rr.mu.Lock()
	defer rr.mu.Unlock()
	rr.downServers = append(rr.downServers, server)
}

func (rr *RoundRobinBalancer) AliveServers() []server.Server {
	rr.mu.Lock()
	defer rr.mu.Unlock()
	return rr.aliveServers
}

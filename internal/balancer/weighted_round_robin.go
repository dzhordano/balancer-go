package balancer

import (
	"sync"

	"github.com/dzhordano/balancer-go/internal/server"
)

type WeightedRoundRobinBalancer struct {
	downServers  []server.Server
	aliveServers []server.Server
	index        int
	current      int // Nth request number
	mu           sync.Mutex
}

func (wrr *WeightedRoundRobinBalancer) SetServers(servers []server.Server) {
	wrr.aliveServers = servers

	wrr.index = 0
	wrr.current = 1
}

func (wrr *WeightedRoundRobinBalancer) SelectServer(args ...interface{}) *server.Server {
	wrr.mu.Lock()
	defer wrr.mu.Unlock()

	if len(wrr.aliveServers) == 0 {
		return nil
	}

	server := &wrr.aliveServers[wrr.index]
	if wrr.current >= server.Weight {
		wrr.index = (wrr.index + 1) % len(wrr.aliveServers)
		wrr.current = 1
	} else {
		wrr.current++
	}

	return server
}

func (wrr *WeightedRoundRobinBalancer) DownServers() []server.Server {
	wrr.mu.Lock()
	defer wrr.mu.Unlock()
	return wrr.downServers
}

func (wrr *WeightedRoundRobinBalancer) RemoveAliveServer(index int) {
	wrr.mu.Lock()
	defer wrr.mu.Unlock()

	wrr.aliveServers = append(wrr.aliveServers[:index], wrr.aliveServers[index+1:]...)
	if len(wrr.aliveServers) == 0 {
		wrr.index = 0
		return
	}
	wrr.index = (wrr.index) % len(wrr.aliveServers)
}

func (wrr *WeightedRoundRobinBalancer) RemoveDownServer(index int) {
	wrr.mu.Lock()
	defer wrr.mu.Unlock()
	wrr.downServers = append(wrr.downServers[:index], wrr.downServers[index+1:]...)
}

func (wrr *WeightedRoundRobinBalancer) AddAliveServer(server server.Server) {
	wrr.mu.Lock()
	defer wrr.mu.Unlock()
	wrr.aliveServers = append(wrr.aliveServers, server)
}

func (wrr *WeightedRoundRobinBalancer) AddDownServer(server server.Server) {
	wrr.mu.Lock()
	defer wrr.mu.Unlock()
	wrr.downServers = append(wrr.downServers, server)
}

func (wrr *WeightedRoundRobinBalancer) AliveServers() []server.Server {
	wrr.mu.Lock()
	defer wrr.mu.Unlock()
	return wrr.aliveServers
}

package balancer

import (
	"sync"

	"github.com/dzhordano/balancer-go/internal/server"
)

type LeastConnectionsBalancer struct {
	mu           sync.Mutex
	downServers  []server.Server
	aliveServers []server.Server
}

func (lc *LeastConnectionsBalancer) SetServers(servers []server.Server) {
	lc.aliveServers = servers
}

func (lc *LeastConnectionsBalancer) SelectServer(args ...interface{}) *server.Server {
	lc.mu.Lock()
	defer lc.mu.Unlock()

	if len(lc.aliveServers) == 0 {
		return nil
	}

	minConns := int(^uint(0) >> 1) // Max Int
	var server *server.Server

	for _, srv := range lc.aliveServers {
		currConns := int(srv.CurrentConnections())

		if currConns < minConns {
			minConns = currConns
			server = &srv
		}
	}

	return server
}

func (lc *LeastConnectionsBalancer) DownServers() []server.Server {
	lc.mu.Lock()
	defer lc.mu.Unlock()
	return lc.downServers
}

func (lc *LeastConnectionsBalancer) RemoveAliveServer(index int) {
	lc.mu.Lock()
	defer lc.mu.Unlock()
	lc.aliveServers = append(lc.aliveServers[:index], lc.aliveServers[index+1:]...)
}

func (lc *LeastConnectionsBalancer) RemoveDownServer(index int) {
	lc.mu.Lock()
	defer lc.mu.Unlock()
	lc.downServers = append(lc.downServers[:index], lc.downServers[index+1:]...)
}

func (lc *LeastConnectionsBalancer) AddAliveServer(server server.Server) {
	lc.mu.Lock()
	defer lc.mu.Unlock()
	lc.aliveServers = append(lc.aliveServers, server)
}

func (lc *LeastConnectionsBalancer) AddDownServer(server server.Server) {
	lc.mu.Lock()
	defer lc.mu.Unlock()
	lc.downServers = append(lc.downServers, server)
}

func (lc *LeastConnectionsBalancer) AliveServers() []server.Server {
	lc.mu.Lock()
	defer lc.mu.Unlock()
	return lc.aliveServers
}

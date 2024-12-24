package handler

import (
	"hash/crc32"
	"math/rand"
	"sync"
)

type Balancer interface {
	SetServers(servers []Server)
	AddAliveServer(server Server)
	AddDownServer(server Server)
	SelectServer(args ...interface{}) *Server
	DownServers() []Server
	AliveServers() []Server
	RemoveAliveServer(index int)
}

type RoundRobinBalancer struct {
	downServers  []Server
	aliveServers []Server
	index        int
	mu           sync.Mutex
}

func (rr *RoundRobinBalancer) SetServers(servers []Server) {
	rr.aliveServers = servers
	rr.index = 0
}

func (rr *RoundRobinBalancer) SelectServer(args ...interface{}) *Server {
	rr.mu.Lock()
	defer rr.mu.Unlock()

	if len(rr.aliveServers) == 0 {
		return nil
	}

	server := &rr.aliveServers[rr.index]
	rr.index = (rr.index + 1) % len(rr.aliveServers)

	return server
}

func (rr *RoundRobinBalancer) DownServers() []Server {
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

func (rr *RoundRobinBalancer) AddAliveServer(server Server) {
	rr.mu.Lock()
	defer rr.mu.Unlock()
	rr.aliveServers = append(rr.aliveServers, server)
}

func (rr *RoundRobinBalancer) AddDownServer(server Server) {
	rr.mu.Lock()
	defer rr.mu.Unlock()
	rr.downServers = append(rr.downServers, server)
}

func (rr *RoundRobinBalancer) AliveServers() []Server {
	rr.mu.Lock()
	defer rr.mu.Unlock()
	return rr.aliveServers
}

type WeightedRoundRobinBalancer struct {
	downServers  []Server
	aliveServers []Server
	index        int
	current      int // Nth request number
	mu           sync.Mutex
}

func (wrr *WeightedRoundRobinBalancer) SetServers(servers []Server) {
	wrr.aliveServers = servers

	wrr.index = 0
	wrr.current = 1
}

func (wrr *WeightedRoundRobinBalancer) SelectServer(args ...interface{}) *Server {
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

func (wrr *WeightedRoundRobinBalancer) DownServers() []Server {
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

func (wrr *WeightedRoundRobinBalancer) AddAliveServer(server Server) {
	wrr.mu.Lock()
	defer wrr.mu.Unlock()
	wrr.aliveServers = append(wrr.aliveServers, server)
}

func (wrr *WeightedRoundRobinBalancer) AddDownServer(server Server) {
	wrr.mu.Lock()
	defer wrr.mu.Unlock()
	wrr.downServers = append(wrr.downServers, server)
}

func (wrr *WeightedRoundRobinBalancer) AliveServers() []Server {
	wrr.mu.Lock()
	defer wrr.mu.Unlock()
	return wrr.aliveServers
}

type LeastConnectionsBalancer struct {
	mu           sync.Mutex
	downServers  []Server
	aliveServers []Server
}

func (lc *LeastConnectionsBalancer) SetServers(servers []Server) {
	lc.aliveServers = servers
}

func (lc *LeastConnectionsBalancer) SelectServer(args ...interface{}) *Server {
	lc.mu.Lock()
	defer lc.mu.Unlock()

	if len(lc.aliveServers) == 0 {
		return nil
	}

	minConns := int(^uint(0) >> 1) // Max Int
	var server *Server

	for _, srv := range lc.aliveServers {
		currConns := int(srv.CurrentConnections())

		if currConns < minConns {
			minConns = currConns
			server = &srv
		}
	}

	server.IncrementConnections()

	return server
}

func (lc *LeastConnectionsBalancer) DownServers() []Server {
	lc.mu.Lock()
	defer lc.mu.Unlock()
	return lc.downServers
}

func (lc *LeastConnectionsBalancer) RemoveAliveServer(index int) {
	lc.mu.Lock()
	defer lc.mu.Unlock()
	lc.aliveServers = append(lc.aliveServers[:index], lc.aliveServers[index+1:]...)
}

func (lc *LeastConnectionsBalancer) AddAliveServer(server Server) {
	lc.mu.Lock()
	defer lc.mu.Unlock()
	lc.aliveServers = append(lc.aliveServers, server)
}

func (lc *LeastConnectionsBalancer) AddDownServer(server Server) {
	lc.mu.Lock()
	defer lc.mu.Unlock()
	lc.downServers = append(lc.downServers, server)
}

func (lc *LeastConnectionsBalancer) AliveServers() []Server {
	lc.mu.Lock()
	defer lc.mu.Unlock()
	return lc.aliveServers
}

type HashBalancer struct {
	downServers  []Server
	aliveServers []Server
	mu           sync.Mutex
}

func (hb *HashBalancer) SetServers(servers []Server) {
	hb.mu.Lock()
	defer hb.mu.Unlock()
	hb.aliveServers = servers
}

func (hb *HashBalancer) SelectServer(args ...interface{}) *Server {
	hb.mu.Lock()
	defer hb.mu.Unlock()

	if len(hb.aliveServers) == 0 {
		return nil
	}

	if len(args) < 1 {
		return nil // FIXME Требуется ключ
	}

	key, ok := args[0].(string)
	if !ok {
		return nil
	}

	hash := crc32.ChecksumIEEE([]byte(key))
	index := int(hash) % len(hb.aliveServers)

	return &hb.aliveServers[index]
}

func (hb *HashBalancer) DownServers() []Server {
	hb.mu.Lock()
	defer hb.mu.Unlock()
	return hb.downServers
}

func (hb *HashBalancer) RemoveAliveServer(index int) {
	hb.mu.Lock()
	defer hb.mu.Unlock()
	hb.aliveServers = append(hb.aliveServers[:index], hb.aliveServers[index+1:]...)
}

func (hb *HashBalancer) AddAliveServer(server Server) {
	hb.mu.Lock()
	defer hb.mu.Unlock()
	hb.aliveServers = append(hb.aliveServers, server)
}

func (hb *HashBalancer) AddDownServer(server Server) {
	hb.mu.Lock()
	defer hb.mu.Unlock()
	hb.downServers = append(hb.downServers, server)
}

func (hb *HashBalancer) AliveServers() []Server {
	hb.mu.Lock()
	defer hb.mu.Unlock()
	return hb.aliveServers
}

// TODO impl random
type RandomBalancer struct {
	downServers  []Server
	aliveServers []Server
	mu           sync.Mutex
}

func (rb *RandomBalancer) SetServers(servers []Server) {
	rb.mu.Lock()
	defer rb.mu.Unlock()
	rb.aliveServers = servers
}

func (rb *RandomBalancer) SelectServer(args ...interface{}) *Server {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	if len(rb.aliveServers) == 0 {
		return nil
	}

	return &rb.aliveServers[rand.Intn(len(rb.aliveServers))]
}

func (rb *RandomBalancer) DownServers() []Server {
	rb.mu.Lock()
	defer rb.mu.Unlock()
	return rb.downServers
}

func (rb *RandomBalancer) AliveServers() []Server {
	rb.mu.Lock()
	defer rb.mu.Unlock()
	return rb.aliveServers
}

func (rb *RandomBalancer) RemoveAliveServer(index int) {
	rb.mu.Lock()
	defer rb.mu.Unlock()
	rb.aliveServers = append(rb.aliveServers[:index], rb.aliveServers[index+1:]...)
}

func (rb *RandomBalancer) AddAliveServer(server Server) {
	rb.mu.Lock()
	defer rb.mu.Unlock()
	rb.aliveServers = append(rb.aliveServers, server)
}

func (rb *RandomBalancer) AddDownServer(server Server) {
	rb.mu.Lock()
	defer rb.mu.Unlock()
	rb.downServers = append(rb.downServers, server)
}

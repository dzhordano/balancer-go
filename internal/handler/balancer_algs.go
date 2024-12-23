package handler

import (
	"fmt"
	"hash/crc32"
	"log/slog"
	"math/rand"
	"net/http"
	"sync"
	"time"
)

type Balancer interface {
	SetServers(servers []Server)
	AddAliveServer(server Server)
	SelectServer(args ...interface{}) *Server
	Servers() []Server
	AliveServers() []Server
	RemoveServer(index int)
}

type RoundRobinBalancer struct {
	servers      []Server
	aliveServers []Server
	index        int
	mu           sync.Mutex
}

func (rr *RoundRobinBalancer) SetServers(servers []Server) {
	rr.servers = servers
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

func (rr *RoundRobinBalancer) Servers() []Server {
	rr.mu.Lock()
	defer rr.mu.Unlock()
	return rr.servers
}

func (rr *RoundRobinBalancer) RemoveServer(index int) {
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

func (rr *RoundRobinBalancer) AliveServers() []Server {
	rr.mu.Lock()
	defer rr.mu.Unlock()
	return rr.aliveServers
}

type WeightedRoundRobinBalancer struct {
	servers      []Server
	aliveServers []Server
	index        int
	current      int // Nth request number
	mu           sync.Mutex
}

func (wrr *WeightedRoundRobinBalancer) SetServers(servers []Server) {
	wrr.servers = servers
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

func (wrr *WeightedRoundRobinBalancer) Servers() []Server {
	wrr.mu.Lock()
	defer wrr.mu.Unlock()
	return wrr.servers
}

func (wrr *WeightedRoundRobinBalancer) RemoveServer(index int) {
	wrr.mu.Lock()
	defer wrr.mu.Unlock()
	wrr.servers = append(wrr.servers[:index], wrr.servers[index+1:]...)
	if len(wrr.servers) == 0 {
		wrr.index = 0
		return
	}
	wrr.index = (wrr.index) % len(wrr.servers)
}

func (wrr *WeightedRoundRobinBalancer) AddAliveServer(server Server) {
	wrr.mu.Lock()
	defer wrr.mu.Unlock()
	wrr.aliveServers = append(wrr.aliveServers, server)
}

func (wrr *WeightedRoundRobinBalancer) AliveServers() []Server {
	wrr.mu.Lock()
	defer wrr.mu.Unlock()
	return wrr.aliveServers
}

func (rr *RoundRobinBalancer) HealthCheck(interval time.Duration, timeout time.Duration) {
	time.Sleep(interval)

	var downServers []Server

	for {
		if len(rr.AliveServers()) == 0 {
			return
		}

		for i := range rr.aliveServers {
			start := time.Now()
			resp, err := http.Get(fmt.Sprintf("http://%s/health", rr.servers[i].URL))
			if err != nil {
				slog.Info("failed to get health check", slog.String("server", rr.servers[i].URL), slog.String("error", err.Error()))

				rr.mu.Lock()
				downServers = append(downServers, rr.aliveServers[i])
				rr.mu.Unlock()

				rr.RemoveServer(i)
				break
			}
			resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				slog.Info("server is not alive", slog.String("server", rr.servers[i].URL), slog.Int("status_code", resp.StatusCode))

				rr.mu.Lock()
				downServers = append(downServers, rr.aliveServers[i])
				rr.mu.Unlock()

				rr.RemoveServer(i)
				break
			}

			elapsed := time.Since(start)
			if elapsed > timeout {
				slog.Info("server is not alive", slog.String("server", rr.servers[i].URL), slog.Duration("elapsed", elapsed))

				rr.mu.Lock()
				downServers = append(downServers, rr.aliveServers[i])
				rr.mu.Unlock()

				rr.RemoveServer(i)
				break
			}
		}

		if len(downServers) > 0 {
			for i := range downServers {
				start := time.Now()
				resp, err := http.Get(fmt.Sprintf("http://%s/health", downServers[i].URL))
				if err != nil {
					break
				}
				resp.Body.Close()

				if resp.StatusCode != http.StatusOK {
					break
				}

				elapsed := time.Since(start)
				if elapsed > timeout {
					break
				}

				rr.AddAliveServer(downServers[i])

				rr.mu.Lock()
				downServers = append(downServers[:i], downServers[i+1:]...)
				rr.mu.Unlock()
			}
		}

		time.Sleep(interval)
	}
}

type LeastConnectionsBalancer struct {
	mu           sync.Mutex
	servers      []Server
	aliveServers []Server
	connCount    map[string]int // Хранит количество соединений для каждого сервера
}

func (lc *LeastConnectionsBalancer) SetServers(servers []Server) {
	lc.servers = servers
	lc.aliveServers = servers
	lc.connCount = make(map[string]int, len(servers))
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
		if lc.connCount[srv.URL] < minConns {
			minConns = lc.connCount[srv.URL]
			server = &srv
		}
	}

	// fmt.Println("CURRENT CONNS:", lc.connCount)

	return server
}

func (lc *LeastConnectionsBalancer) Servers() []Server {
	lc.mu.Lock()
	defer lc.mu.Unlock()
	return lc.servers
}

func (lc *LeastConnectionsBalancer) RemoveServer(index int) {
	lc.mu.Lock()
	defer lc.mu.Unlock()
	lc.servers = append(lc.servers[:index], lc.servers[index+1:]...)
}

func (lc *LeastConnectionsBalancer) AddAliveServer(server Server) {
	lc.mu.Lock()
	defer lc.mu.Unlock()
	lc.aliveServers = append(lc.aliveServers, server)
}

func (lc *LeastConnectionsBalancer) AliveServers() []Server {
	lc.mu.Lock()
	defer lc.mu.Unlock()
	return lc.aliveServers
}

type HashBalancer struct {
	servers      []Server
	aliveServers []Server
	mu           sync.Mutex
}

func (hb *HashBalancer) SetServers(servers []Server) {
	hb.mu.Lock()
	defer hb.mu.Unlock()
	hb.servers = servers
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

func (hb *HashBalancer) Servers() []Server {
	hb.mu.Lock()
	defer hb.mu.Unlock()
	return hb.servers
}

func (hb *HashBalancer) RemoveServer(index int) {
	hb.mu.Lock()
	defer hb.mu.Unlock()
	hb.servers = append(hb.servers[:index], hb.servers[index+1:]...)
}

func (hb *HashBalancer) AddAliveServer(server Server) {
	hb.mu.Lock()
	defer hb.mu.Unlock()
	hb.aliveServers = append(hb.aliveServers, server)
}

func (hb *HashBalancer) AliveServers() []Server {
	hb.mu.Lock()
	defer hb.mu.Unlock()
	return hb.aliveServers
}

// TODO impl random
type RandomBalancer struct {
	servers      []Server
	aliveServers []Server
	mu           sync.Mutex
}

func (rb *RandomBalancer) SetServers(servers []Server) {
	rb.mu.Lock()
	defer rb.mu.Unlock()
	rb.servers = servers
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

func (rb *RandomBalancer) Servers() []Server {
	rb.mu.Lock()
	defer rb.mu.Unlock()
	return rb.servers
}

func (rb *RandomBalancer) AliveServers() []Server {
	rb.mu.Lock()
	defer rb.mu.Unlock()
	return rb.aliveServers
}

func (rb *RandomBalancer) RemoveServer(index int) {
	rb.mu.Lock()
	defer rb.mu.Unlock()
	rb.servers = append(rb.servers[:index], rb.servers[index+1:]...)
}

func (rb *RandomBalancer) AddAliveServer(server Server) {
	rb.mu.Lock()
	defer rb.mu.Unlock()
	rb.aliveServers = append(rb.aliveServers, server)
}

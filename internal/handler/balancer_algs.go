package handler

import (
	"fmt"
	"hash/crc32"
	"log/slog"
	"net/http"
	"sync"
	"time"
)

type Balancer interface {
	SetServers(servers []Server)
	SelectServer(args ...interface{}) *Server
	Servers() []Server
	RemoveServer(index int)
	HealthCheck(interval time.Duration, timeout time.Duration)
}

type RoundRobinBalancer struct {
	servers []Server
	index   int
	mu      sync.Mutex
}

func (rr *RoundRobinBalancer) SetServers(servers []Server) {
	rr.servers = servers
	rr.index = 0
}

func (rr *RoundRobinBalancer) SelectServer(args ...interface{}) *Server {
	rr.mu.Lock()
	defer rr.mu.Unlock()

	if len(rr.servers) == 0 {
		return nil
	}

	server := &rr.servers[rr.index]
	rr.index = (rr.index + 1) % len(rr.servers)

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
	rr.servers = append(rr.servers[:index], rr.servers[index+1:]...)
	if len(rr.servers) == 0 {
		rr.index = 0
		return
	}
	rr.index = (rr.index) % len(rr.servers)
}

func (rr *RoundRobinBalancer) HealthCheck(interval time.Duration, timeout time.Duration) {
	time.Sleep(interval)

	for {
		servers := rr.Servers()
		if len(servers) == 0 {
			return
		}

		for i := range servers {
			start := time.Now()
			resp, err := http.Get(fmt.Sprintf("http://%s/health", rr.servers[i].URL))
			if err != nil {
				slog.Error("failed to get health", slog.Any("server", rr.servers[i].URL), slog.Any("error", err))

				rr.RemoveServer(i)
				break
			}
			resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				slog.Error("server is not alive", slog.Any("server", rr.servers[i].URL), slog.Any("status_code", resp.StatusCode))

				rr.RemoveServer(i)
				break
			}

			elapsed := time.Since(start)
			if elapsed > timeout {
				slog.Error("server is not alive", slog.Any("server", rr.servers[i].URL), slog.Any("elapsed", elapsed))

				rr.RemoveServer(i)
				break
			}

			fmt.Println("ALIVE", rr.servers[i].URL)
		}

		time.Sleep(interval)
	}
}

type WeightedRoundRobinBalancer struct {
	servers []Server
	index   int
	current int // Nth request number
	mu      sync.Mutex
}

func (wrr *WeightedRoundRobinBalancer) SetServers(servers []Server) {
	wrr.servers = servers

	wrr.index = 0
	wrr.current = 1
}

func (wrr *WeightedRoundRobinBalancer) SelectServer(args ...interface{}) *Server {
	wrr.mu.Lock()
	defer wrr.mu.Unlock()

	if len(wrr.servers) == 0 {
		return nil
	}

	server := &wrr.servers[wrr.index]
	if wrr.current >= server.Weight {
		wrr.index = (wrr.index + 1) % len(wrr.servers)
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

func (wrr *WeightedRoundRobinBalancer) HealthCheck(interval time.Duration, timeout time.Duration) {
	time.Sleep(interval)

	for {
		servers := wrr.Servers()
		if len(servers) == 0 {
			return
		}

		for i := range servers {
			start := time.Now()
			resp, err := http.Get(fmt.Sprintf("http://%s/health", wrr.servers[i].URL))
			if err != nil {
				slog.Error("failed to get health", slog.Any("server", wrr.servers[i].URL), slog.Any("error", err))

				wrr.RemoveServer(i)
				break
			}
			resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				slog.Error("server is not alive", slog.Any("server", wrr.servers[i].URL), slog.Any("status_code", resp.StatusCode))

				wrr.RemoveServer(i)
				break
			}

			elapsed := time.Since(start)
			if elapsed > timeout {
				slog.Error("server is not alive", slog.Any("server", wrr.servers[i].URL), slog.Any("elapsed", elapsed))

				wrr.RemoveServer(i)
				break
			}

			fmt.Println("ALIVE", wrr.servers[i].URL)
		}

		time.Sleep(interval)
	}
}

// FIXME
// type LeastConnectionsBalancer struct {
// 	mu        *sync.Mutex
// 	connCount map[string]int // Хранит количество соединений для каждого сервера
// }

// func NewLeastConnectionsBalancer(args ...interface{}) *LeastConnectionsBalancer {
// 	return &LeastConnectionsBalancer{
// 		mu:        &sync.Mutex{},
// 		connCount: make(map[string]int)}
// }

// func (lc *LeastConnectionsBalancer) SelectServer(args ...interface{}) *Server {
// 	lc.mu.Lock()
// 	defer lc.mu.Unlock()

// 	aliveServers := getAliveServers(servers)
// 	if len(aliveServers) == 0 {
// 		return nil
// 	}

// 	servers = &aliveServers

// 	var selected *Server
// 	minConnections := int(^uint(0) >> 1) // Max Int

// 	for _, srv := range aliveServers {
// 		if lc.connCount[srv.URL] < minConnections {
// 			minConnections = lc.connCount[srv.URL]
// 			selected = &srv
// 		}
// 	}

// 	if selected != nil {
// 		lc.connCount[selected.URL]++
// 	}
// 	return selected
// }

type HashBalancer struct {
	servers []Server
	mu      sync.Mutex
}

func (hb *HashBalancer) SetServers(servers []Server) {
	hb.mu.Lock()
	defer hb.mu.Unlock()
	hb.servers = servers
}

func (hb *HashBalancer) SelectServer(args ...interface{}) *Server {
	hb.mu.Lock()
	defer hb.mu.Unlock()

	if len(hb.servers) == 0 {
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
	index := int(hash) % len(hb.servers)

	return &hb.servers[index]
}

// TODO impl random

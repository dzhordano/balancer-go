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
	rr.mu.Lock()
	defer rr.mu.Unlock()
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

// type WeightedRoundRobinBalancer struct {
// 	servers []Server
// 	weights []int
// 	index   int
// 	current int
// 	mu      sync.Mutex
// }

// func (wrr *WeightedRoundRobinBalancer) SetServers(servers []Server) {
// 	wrr.mu.Lock()
// 	defer wrr.mu.Unlock()

// 	aliveServers := getAliveServers(servers)
// 	wrr.servers = aliveServers

// 	wrr.weights = make([]int, len(aliveServers))
// 	for i := range aliveServers {
// 		wrr.weights[i] = aliveServers[i].Weight
// 	}

// 	wrr.index = 0
// 	wrr.current = 0
// }

// func (wrr *WeightedRoundRobinBalancer) SelectServer(args ...interface{}) *Server {
// 	wrr.mu.Lock()
// 	defer wrr.mu.Unlock()

// 	if len(wrr.servers) == 0 {
// 		return nil
// 	}

// 	for {
// 		wrr.index = (wrr.index + 1) % len(wrr.servers)
// 		if wrr.index == 0 {
// 			wrr.current--
// 			if wrr.current <= 0 {
// 				wrr.current = wrr.weights[wrr.index]
// 			}
// 		}

// 		if wrr.weights[wrr.index] >= wrr.current {
// 			return &wrr.servers[wrr.index]
// 		}
// 	}
// }

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

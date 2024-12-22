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
				slog.Info("failed to get health check", slog.String("server", rr.servers[i].URL), slog.String("error", err.Error()))

				rr.RemoveServer(i)
				break
			}
			resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				slog.Info("server is not alive", slog.String("server", rr.servers[i].URL), slog.Int("status_code", resp.StatusCode))

				rr.RemoveServer(i)
				break
			}

			elapsed := time.Since(start)
			if elapsed > timeout {
				slog.Info("server is not alive", slog.String("server", rr.servers[i].URL), slog.Duration("elapsed", elapsed))

				rr.RemoveServer(i)
				break
			}
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
				slog.Info("failed to get health check", slog.String("server", wrr.servers[i].URL), slog.String("error", err.Error()))

				wrr.RemoveServer(i)
				break
			}
			resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				slog.Info("server is not alive", slog.String("server", wrr.servers[i].URL), slog.Int("status_code", resp.StatusCode))

				wrr.RemoveServer(i)
				break
			}

			elapsed := time.Since(start)
			if elapsed > timeout {
				slog.Info("server is not alive", slog.String("server", wrr.servers[i].URL), slog.Duration("elapsed", elapsed))

				wrr.RemoveServer(i)
				break
			}
		}

		time.Sleep(interval)
	}
}

type LeastConnectionsBalancer struct {
	mu        sync.Mutex
	servers   []Server
	connCount map[string]int // Хранит количество соединений для каждого сервера
}

func (lc *LeastConnectionsBalancer) SetServers(servers []Server) {
	lc.servers = servers
	lc.connCount = make(map[string]int, len(servers))
}

func (lc *LeastConnectionsBalancer) SelectServer(args ...interface{}) *Server {
	lc.mu.Lock()
	defer lc.mu.Unlock()

	if len(lc.servers) == 0 {
		return nil
	}

	minConns := int(^uint(0) >> 1) // Max Int
	var server *Server

	for _, srv := range lc.servers {
		if lc.connCount[srv.URL] < minConns {
			minConns = lc.connCount[srv.URL]
			server = &srv
		}
	}

	fmt.Println("CURRENT CONNS:", lc.connCount)

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

func (lc *LeastConnectionsBalancer) HealthCheck(interval time.Duration, timeout time.Duration) {
	time.Sleep(interval)

	for {
		servers := lc.Servers()
		if len(servers) == 0 {
			return
		}

		for i := range servers {
			start := time.Now()
			resp, err := http.Get(fmt.Sprintf("http://%s/health", lc.servers[i].URL))
			if err != nil {
				slog.Info("failed to get health check", slog.String("server", lc.servers[i].URL), slog.String("error", err.Error()))

				lc.RemoveServer(i)
				break
			}
			resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				slog.Info("server is not alive", slog.String("server", lc.servers[i].URL), slog.Int("status_code", resp.StatusCode))

				lc.RemoveServer(i)
				break
			}

			elapsed := time.Since(start)
			if elapsed > timeout {
				slog.Info("server is not alive", slog.String("server", lc.servers[i].URL), slog.Duration("elapsed", elapsed))

				lc.RemoveServer(i)
				break
			}
		}

		time.Sleep(interval)
	}
}

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

func (hb *HashBalancer) HealthCheck(interval time.Duration, timeout time.Duration) {
	time.Sleep(interval)

	for {
		servers := hb.Servers()
		if len(servers) == 0 {
			return
		}

		for i := range servers {
			start := time.Now()
			resp, err := http.Get(fmt.Sprintf("http://%s/health", hb.servers[i].URL))
			if err != nil {
				slog.Info("failed to get health check", slog.String("server", hb.servers[i].URL), slog.String("error", err.Error()))

				hb.RemoveServer(i)
				break
			}
			resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				slog.Info("server is not alive", slog.String("server", hb.servers[i].URL), slog.Int("status_code", resp.StatusCode))

				hb.RemoveServer(i)
				break
			}

			elapsed := time.Since(start)
			if elapsed > timeout {
				slog.Info("server is not alive", slog.String("server", hb.servers[i].URL), slog.Duration("elapsed", elapsed))

				hb.RemoveServer(i)
				break
			}
		}

		time.Sleep(interval)
	}
}

// TODO impl random
type RandomBalancer struct {
	servers []Server
	mu      sync.Mutex
}

func (hb *RandomBalancer) SetServers(servers []Server) {
	hb.mu.Lock()
	defer hb.mu.Unlock()
	hb.servers = servers
}

func (hb *RandomBalancer) SelectServer(args ...interface{}) *Server {
	hb.mu.Lock()
	defer hb.mu.Unlock()

	if len(hb.servers) == 0 {
		return nil
	}

	return &hb.servers[rand.Intn(len(hb.servers))]
}

func (hb *RandomBalancer) Servers() []Server {
	hb.mu.Lock()
	defer hb.mu.Unlock()
	return hb.servers
}

func (hb *RandomBalancer) RemoveServer(index int) {
	hb.mu.Lock()
	defer hb.mu.Unlock()
	hb.servers = append(hb.servers[:index], hb.servers[index+1:]...)
}

func (hb *RandomBalancer) HealthCheck(interval time.Duration, timeout time.Duration) {
	time.Sleep(interval)

	for {
		servers := hb.Servers()
		if len(servers) == 0 {
			return
		}

		for i := range servers {
			start := time.Now()
			resp, err := http.Get(fmt.Sprintf("http://%s/health", hb.servers[i].URL))
			if err != nil {
				slog.Info("failed to get health check", slog.String("server", hb.servers[i].URL), slog.String("error", err.Error()))

				hb.RemoveServer(i)
				break
			}
			resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				slog.Info("server is not alive", slog.String("server", hb.servers[i].URL), slog.Int("status_code", resp.StatusCode))

				hb.RemoveServer(i)
				break
			}

			elapsed := time.Since(start)
			if elapsed > timeout {
				slog.Info("server is not alive", slog.String("server", hb.servers[i].URL), slog.Duration("elapsed", elapsed))

				hb.RemoveServer(i)
				break
			}
		}

		time.Sleep(interval)
	}
}

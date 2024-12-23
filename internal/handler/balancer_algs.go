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
	HealthCheck(interval time.Duration, timeout time.Duration)
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

func (rr *RoundRobinBalancer) HealthCheck(interval time.Duration, timeout time.Duration) {
	time.Sleep(interval)

	for {
		if len(rr.Servers()) == 0 {
			return
		}

		for i := range rr.servers {
			start := time.Now()
			resp, err := http.Get(fmt.Sprintf("http://%s/health", rr.servers[i].URL))
			if err != nil {
				slog.Info("failed to get health check", slog.String("server", rr.servers[i].URL), slog.String("error", err.Error()))

				rr.servers[i].Alive = false
				rr.RemoveServer(i)
				break
			}
			resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				slog.Info("server is not alive", slog.String("server", rr.servers[i].URL), slog.Int("status_code", resp.StatusCode))

				rr.servers[i].Alive = false
				rr.RemoveServer(i)
				break
			}

			elapsed := time.Since(start)
			if elapsed > timeout {
				slog.Info("server is not alive", slog.String("server", rr.servers[i].URL), slog.Duration("elapsed", elapsed))

				rr.servers[i].Alive = false
				rr.RemoveServer(i)
				break
			}

			slog.Debug("server is alive", slog.String("server", rr.servers[i].URL), slog.Duration("elapsed", elapsed))
			if !rr.servers[i].Alive {
				rr.servers[i].Alive = true
				rr.AddAliveServer(rr.servers[i])
			}
		}

		time.Sleep(interval)
	}
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

func (wrr *WeightedRoundRobinBalancer) HealthCheck(interval time.Duration, timeout time.Duration) {
	time.Sleep(interval)

	for {
		if len(wrr.Servers()) == 0 {
			return
		}

		for i := range wrr.servers {
			start := time.Now()
			resp, err := http.Get(fmt.Sprintf("http://%s/health", wrr.servers[i].URL))
			if err != nil {
				slog.Info("failed to get health check", slog.String("server", wrr.servers[i].URL), slog.String("error", err.Error()))

				wrr.servers[i].Alive = false
				wrr.RemoveServer(i)
				break
			}
			resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				slog.Info("server is not alive", slog.String("server", wrr.servers[i].URL), slog.Int("status_code", resp.StatusCode))

				wrr.servers[i].Alive = false
				wrr.RemoveServer(i)
				break
			}

			elapsed := time.Since(start)
			if elapsed > timeout {
				slog.Info("server is not alive", slog.String("server", wrr.servers[i].URL), slog.Duration("elapsed", elapsed))

				wrr.servers[i].Alive = false
				wrr.RemoveServer(i)
				break
			}

			slog.Debug("server is alive", slog.String("server", wrr.servers[i].URL), slog.Duration("elapsed", elapsed))
			if !wrr.servers[i].Alive {
				wrr.servers[i].Alive = true
				wrr.AddAliveServer(wrr.servers[i])
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

func (lc *LeastConnectionsBalancer) HealthCheck(interval time.Duration, timeout time.Duration) {
	time.Sleep(interval)

	for {
		if len(lc.Servers()) == 0 {
			return
		}

		for i := range lc.servers {
			start := time.Now()
			resp, err := http.Get(fmt.Sprintf("http://%s/health", lc.servers[i].URL))
			if err != nil {
				slog.Info("failed to get health check", slog.String("server", lc.servers[i].URL), slog.String("error", err.Error()))

				lc.servers[i].Alive = false
				lc.RemoveServer(i)
				break
			}
			resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				slog.Info("server is not alive", slog.String("server", lc.servers[i].URL), slog.Int("status_code", resp.StatusCode))

				lc.servers[i].Alive = false
				lc.RemoveServer(i)
				break
			}

			elapsed := time.Since(start)
			if elapsed > timeout {
				slog.Info("server is not alive", slog.String("server", lc.servers[i].URL), slog.Duration("elapsed", elapsed))

				lc.servers[i].Alive = false
				lc.RemoveServer(i)
				break
			}

			slog.Debug("server is alive", slog.String("server", lc.servers[i].URL), slog.Duration("elapsed", elapsed))
			if !lc.servers[i].Alive {
				lc.servers[i].Alive = true
				lc.AddAliveServer(lc.servers[i])
			}
		}

		time.Sleep(interval)
	}
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

func (hb *HashBalancer) HealthCheck(interval time.Duration, timeout time.Duration) {
	time.Sleep(interval)

	for {
		if len(hb.Servers()) == 0 {
			return
		}

		for i := range hb.servers {
			start := time.Now()
			resp, err := http.Get(fmt.Sprintf("http://%s/health", hb.servers[i].URL))
			if err != nil {
				slog.Info("failed to get health check", slog.String("server", hb.servers[i].URL), slog.String("error", err.Error()))

				hb.servers[i].Alive = false
				hb.RemoveServer(i)
				break
			}
			resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				slog.Info("server is not alive", slog.String("server", hb.servers[i].URL), slog.Int("status_code", resp.StatusCode))

				hb.servers[i].Alive = false
				hb.RemoveServer(i)
				break
			}

			elapsed := time.Since(start)
			if elapsed > timeout {
				slog.Info("server is not alive", slog.String("server", hb.servers[i].URL), slog.Duration("elapsed", elapsed))

				hb.servers[i].Alive = false
				hb.RemoveServer(i)
				break
			}

			slog.Debug("server is alive", slog.String("server", hb.servers[i].URL), slog.Duration("elapsed", elapsed))
			if !hb.servers[i].Alive {
				hb.servers[i].Alive = true
				hb.AddAliveServer(hb.servers[i])
			}
		}

		time.Sleep(interval)
	}
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

func (rb *RandomBalancer) HealthCheck(interval time.Duration, timeout time.Duration) {
	time.Sleep(interval)

	for {
		if len(rb.Servers()) == 0 {
			return
		}

		for i := range rb.servers {
			start := time.Now()
			resp, err := http.Get(fmt.Sprintf("http://%s/health", rb.servers[i].URL))
			if err != nil {
				slog.Info("failed to get health check", slog.String("server", rb.servers[i].URL), slog.String("error", err.Error()))

				rb.servers[i].Alive = false
				rb.RemoveServer(i)
				break
			}
			resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				slog.Info("server is not alive", slog.String("server", rb.servers[i].URL), slog.Int("status_code", resp.StatusCode))

				rb.servers[i].Alive = false
				rb.RemoveServer(i)
				break
			}

			elapsed := time.Since(start)
			if elapsed > timeout {
				slog.Info("server is not alive", slog.String("server", rb.servers[i].URL), slog.Duration("elapsed", elapsed))

				rb.servers[i].Alive = false
				rb.RemoveServer(i)
				break
			}

			slog.Debug("server is alive", slog.String("server", rb.servers[i].URL), slog.Duration("elapsed", elapsed))
			if !rb.servers[i].Alive {
				rb.servers[i].Alive = true
				rb.AddAliveServer(rb.servers[i])
			}
		}

		time.Sleep(interval)
	}
}

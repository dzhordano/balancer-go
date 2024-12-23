package handler

import (
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"
)

type HealthChecker interface {
	HealthCheck(interval time.Duration, timeout time.Duration)
}

type hc struct {
	interval time.Duration
	timeout  time.Duration
	balancer Balancer
	mu       sync.Mutex
}

func NewHealthChecker(interval time.Duration, timeout time.Duration, balancer Balancer) HealthChecker {
	return &hc{interval: interval, timeout: timeout, balancer: balancer}
}

func (hl *hc) HealthCheck(interval time.Duration, timeout time.Duration) {
	time.Sleep(interval)

	var downServers []Server
	servers := hl.balancer.Servers()
	aliveServers := hl.balancer.AliveServers()

	for {
		if len(hl.balancer.AliveServers()) == 0 {
			return
		}

		for i := range aliveServers {
			start := time.Now()
			resp, err := http.Get(fmt.Sprintf("http://%s/health", servers[i].URL))
			if err != nil {
				slog.Info("failed to get health check", slog.String("server", servers[i].URL), slog.String("error", err.Error()))

				hl.mu.Lock()
				downServers = append(downServers, aliveServers[i])
				hl.mu.Unlock()

				hl.balancer.RemoveServer(i)
				break
			}
			resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				slog.Info("server is not alive", slog.String("server", servers[i].URL), slog.Int("status_code", resp.StatusCode))

				hl.mu.Lock()
				downServers = append(downServers, aliveServers[i])
				hl.mu.Unlock()

				hl.balancer.RemoveServer(i)
				break
			}

			elapsed := time.Since(start)
			if elapsed > timeout {
				slog.Info("server is not alive", slog.String("server", servers[i].URL), slog.Duration("elapsed", elapsed))

				hl.mu.Lock()
				downServers = append(downServers, aliveServers[i])
				hl.mu.Unlock()

				hl.balancer.RemoveServer(i)
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

				hl.balancer.AddAliveServer(downServers[i])

				hl.mu.Lock()
				downServers = append(downServers[:i], downServers[i+1:]...)
				hl.mu.Unlock()
			}
		}

		time.Sleep(interval)
	}
}

package handler

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

type HealthChecker interface {
	HealthCheck()
}

type hc struct {
	interval time.Duration
	timeout  time.Duration
	balancer Balancer
}

func NewHealthChecker(interval time.Duration, timeout time.Duration, balancer Balancer) HealthChecker {
	return &hc{
		interval: interval,
		timeout:  timeout,
		balancer: balancer}
}

func (hl *hc) HealthCheck() {
	time.Sleep(hl.interval)

	downServers := hl.balancer.DownServers()
	aliveServers := hl.balancer.AliveServers()

	for {
		if len(hl.balancer.AliveServers()) == 0 {
			return
		}

		for i := range aliveServers {
			start := time.Now()
			resp, err := http.Get(fmt.Sprintf("http://%s/health", aliveServers[i].URL))
			if err != nil {
				slog.Info("failed to get health check", slog.String("server", aliveServers[i].URL), slog.String("error", err.Error()))

				hl.balancer.AddDownServer(aliveServers[i])
				hl.balancer.RemoveAliveServer(i)

				break
			}
			resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				slog.Info("server is not alive", slog.String("server", aliveServers[i].URL), slog.Int("status_code", resp.StatusCode))

				hl.balancer.AddDownServer(aliveServers[i])
				hl.balancer.RemoveAliveServer(i)
				break
			}

			elapsed := time.Since(start)
			if elapsed > hl.timeout {
				slog.Info("server is not alive", slog.String("server", aliveServers[i].URL), slog.Duration("elapsed", elapsed))

				hl.balancer.AddDownServer(aliveServers[i])
				hl.balancer.RemoveAliveServer(i)
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
				if elapsed > hl.timeout {
					break
				}

				hl.balancer.AddAliveServer(downServers[i])
			}
		}

		time.Sleep(hl.interval)
	}
}

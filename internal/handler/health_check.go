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
	log      *slog.Logger
	interval time.Duration
	timeout  time.Duration
	balancer Balancer
}

func NewHealthChecker(logger *slog.Logger, interval time.Duration, timeout time.Duration, balancer Balancer) HealthChecker {
	return &hc{
		log:      logger,
		interval: interval,
		timeout:  timeout,
		balancer: balancer}
}

func (hl *hc) HealthCheck() {
	time.Sleep(hl.interval)

	for {
		downServers := hl.balancer.DownServers()
		aliveServers := hl.balancer.AliveServers()

		if len(hl.balancer.AliveServers()) == 0 {
			return
		}

		for i := range aliveServers {
			start := time.Now()
			resp, err := http.Get(fmt.Sprintf("http://%s/health", aliveServers[i].URL))
			if err != nil {
				hl.log.Info("HEALTHCHECK: failed to get health check", slog.String("server", aliveServers[i].URL), slog.String("error", err.Error()))

				hl.balancer.AddDownServer(aliveServers[i])
				hl.balancer.RemoveAliveServer(i)

				continue
			}
			resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				hl.log.Info("HEALTHCHECK: server is not alive", slog.String("server", aliveServers[i].URL), slog.Int("status_code", resp.StatusCode))

				hl.balancer.AddDownServer(aliveServers[i])
				hl.balancer.RemoveAliveServer(i)

				continue
			}

			elapsed := time.Since(start)
			if elapsed > hl.timeout {
				hl.log.Info("HEALTHCHECK: server is not alive", slog.String("server", aliveServers[i].URL), slog.Duration("elapsed", elapsed))

				hl.balancer.AddDownServer(aliveServers[i])
				hl.balancer.RemoveAliveServer(i)

				continue
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

				hl.log.Info("HEALTHCHECK: server is alive", slog.String("server", downServers[i].URL), slog.Duration("elapsed", elapsed))

				hl.balancer.AddAliveServer(downServers[i])
			}
		}

		time.Sleep(hl.interval)
	}

}

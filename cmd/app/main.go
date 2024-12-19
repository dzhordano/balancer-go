package main

import (
	"context"
	"log/slog"
	"math/rand/v2"
	"net"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/dzhordano/balancer-go/internal/config"
	"github.com/dzhordano/balancer-go/internal/handler"
	"github.com/dzhordano/balancer-go/internal/server"
	"github.com/dzhordano/balancer-go/pkg/logger"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	cfg := config.NewConfig()

	log := logger.NewSlogLogger(
		os.Stdout,
		cfg.Logging.Level,
	)

	mainWG := sync.WaitGroup{}

	for i := range cfg.Servers {
		mainWG.Add(1)
		go func() {
			defer mainWG.Done()

			log.Info("starting http server", slog.Int("server number", i+1), slog.String("server url", cfg.Servers[i].URL))

			newSrv := server.NewHTTPServer(cfg.Servers[i].URL, handler.DefaultRoutes())

			go func() {
				time.Sleep(time.Duration(rand.IntN(30)) * time.Second)
				newSrv.Shutdown(context.Background())
			}()

			if err := newSrv.Run(); err != nil {
				log.Error("error runnning http server",
					slog.String("server url", cfg.Servers[i].URL),
					slog.String("error", err.Error()))
			}
		}()
	}

	servers := make([]handler.Server, len(cfg.Servers))
	for i := range cfg.Servers {
		servers[i] = *handler.NewServer(
			cfg.Servers[i].URL,
			cfg.Servers[i].Weight,
		)
	}

	srv := server.NewHTTPServer(
		net.JoinHostPort(cfg.HTTPServer.Host, cfg.HTTPServer.Port),
		handler.NewBalancerHandler(log, servers, cfg.BalancingAlg, cfg.HealthCheck.Interval, cfg.HealthCheck.Timeout).Routes(),
	)

	mainWG.Add(1)
	go func() {
		defer mainWG.Done()
		log.Info("starting balancer http server", slog.String("server url", net.JoinHostPort(cfg.HTTPServer.Host, cfg.HTTPServer.Port)))

		if err := srv.Run(); err != nil {
			log.Error("error runnning balancer http server",
				slog.String("server url", net.JoinHostPort(cfg.HTTPServer.Host, cfg.HTTPServer.Port)),
				slog.String("error", err.Error()))
		}
	}()

	go func() {
		http.Handle("/metrics", promhttp.Handler())
		log.Info("starting prometheus metrics endpoint on /metrics")
		http.ListenAndServe(":9091", nil)
	}()

	mainWG.Wait()
}

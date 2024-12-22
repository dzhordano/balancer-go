package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/dzhordano/balancer-go/internal/config"
	"github.com/dzhordano/balancer-go/internal/handler"
	"github.com/dzhordano/balancer-go/internal/server"
	"github.com/dzhordano/balancer-go/pkg/logger"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	flagConfigPath string

	defaultConfigPath = "configs/config.yaml"
)

func main() {
	// Парсинг аргумента для передачи пути к файлу конфигурации.
	//При отсутствии флага устанавлитвается значение по умолчанию (defaultConfigPath).
	flag.StringVar(&flagConfigPath, "c", defaultConfigPath, "path to config file")
	flag.Parse()

	// Инициализация и чтение конфигурации.
	cfg := config.NewConfig(flagConfigPath)

	// Инициализация логгера.
	log := logger.NewSlogLogger(
		os.Stdout,
		cfg.Logging.Level,
	)

	// Создание WaitGroup'ы для синхронизации горутин серверов.
	mainWG := sync.WaitGroup{}

	// Инициализация атомарного значения для хранения времени остановки.
	outageAfter := atomic.Value{}
	outageAfter.Store(cfg.ServersOutage.After)

	// Запуск серверов в отдельных горутинах, которые останавливаются через определенное время.
	// Останока серверов также выполняется в отдельных горутинах для избежания блокировки горутины сервера.
	for i := range cfg.Servers {
		mainWG.Add(1)
		go func() {
			defer mainWG.Done()

			log.Info("starting http server", slog.Int("server number", i+1), slog.String("server url", cfg.Servers[i].URL))

			newSrv := server.NewHTTPServer(cfg.Servers[i].URL, handler.DefaultRoutes())

			go func() {
				if cfg.ServersOutage.After != -1 {
					outAfter := outageAfter.Load().(float64)
					outageAfter.Store(float64(outAfter) * cfg.ServersOutage.Multiplier)
					fmt.Println("server", cfg.Servers[i].URL, "will stop after", time.Duration(outAfter)*time.Second)
					time.Sleep(time.Duration(outAfter) * time.Second)
					newSrv.Shutdown(context.Background())
				}
			}()

			if err := newSrv.Run(); err != nil {
				log.Error("error runnning http server",
					slog.String("server url", cfg.Servers[i].URL),
					slog.String("error", err.Error()))
			}
		}()
	}

	// Заполнение массива доступных серверов для балансировщика.
	servers := make([]handler.Server, len(cfg.Servers))
	for i := range cfg.Servers {
		servers[i] = *handler.NewServer(
			cfg.Servers[i].URL,
			cfg.Servers[i].Weight,
		)
	}

	// Инициализация балансировщика.
	srv := server.NewHTTPServer(
		net.JoinHostPort(cfg.HTTPServer.Host, cfg.HTTPServer.Port),
		handler.NewBalancerHandler(log, servers, cfg.BalancingAlg, cfg.HealthCheck.Interval, cfg.HealthCheck.Timeout).Routes(),
	)

	// Запуск балансировщика отдельной горутиной.
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

	// Запуск сервера метрик отдельной горутиной.
	go func() {
		http.Handle("/metrics", promhttp.Handler())
		log.Info("starting prometheus metrics endpoint on /metrics")
		http.ListenAndServe(":9091", nil)
	}()

	// Ожидание завершения всех горутин.
	mainWG.Wait()
}

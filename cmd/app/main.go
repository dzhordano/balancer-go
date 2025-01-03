package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/dzhordano/balancer-go/internal/balancer"
	"github.com/dzhordano/balancer-go/internal/config"
	"github.com/dzhordano/balancer-go/internal/healthcheck"
	"github.com/dzhordano/balancer-go/internal/httpserver"
	"github.com/dzhordano/balancer-go/internal/routes"
	"github.com/dzhordano/balancer-go/internal/server"
	"github.com/dzhordano/balancer-go/pkg/logger"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	flagConfigPath string
)

func main() {
	// Парсинг аргумента для передачи пути к файлу конфигурации.
	//При отсутствии флага устанавлитвается значение по умолчанию (defaultConfigPath).
	flag.StringVar(&flagConfigPath, "c", "", "path to config file")
	flag.Parse()

	// Инициализация и чтение конфигурации.
	cfg := config.NewConfig(flagConfigPath)

	// Check if path exists, if not - create.
	if _, err := os.Stat(cfg.Logging.Path); err != nil {
		if os.IsNotExist(err) {
			if err := os.MkdirAll(cfg.Logging.Path, 0755); err != nil {
				log.Fatalf("error creating directory: %s", err)
			}
		} else {
			log.Fatalf("error checking path: %s", err)
		}
	}

	// Check if file exists, if not - create.
	if _, err := os.Stat(cfg.Logging.Path + cfg.Logging.File); err != nil {
		if os.IsNotExist(err) {
			if _, err := os.Create(cfg.Logging.Path + cfg.Logging.File); err != nil {
				log.Fatalf("error creating file: %s", err)
			}
		} else {
			log.Fatalf("error checking file: %s", err)
		}
	} else {
		if cfg.Logging.Rewrite {
			if err := os.Remove(cfg.Logging.Path + cfg.Logging.File); err != nil {
				log.Fatalf("error removing file: %s", err)
			}
			if _, err := os.Create(cfg.Logging.Path + cfg.Logging.File); err != nil {
				log.Fatalf("error creating file: %s", err)
			}
		}
	}

	// Файл для хранения логов.
	f, err := os.OpenFile(cfg.Logging.Path+cfg.Logging.File, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalf("error opening file: %s", err)
	}
	defer f.Close()

	// Инициализация логгера.
	logging := logger.NewSlogLogger(
		f,
		cfg.Logging.Level,
	)

	// Создание WaitGroup'ы для синхронизации горутин серверов.
	mainWG := sync.WaitGroup{}

	// Инициализация атомарного значения для хранения времени остановки.
	outageAfter := atomic.Value{}
	outageAfter.Store(cfg.ServersOutage.After)

	// Список всех серверов для последующего закрытия.
	var serversList []*httpserver.HTTPServer
	var startupMU sync.Mutex

	// Запуск серверов в отдельных горутинах, которые останавливаются через определенное время.
	// Останока серверов также выполняется в отдельных горутинах для избежания блокировки горутины сервера.
	for i := range cfg.Servers {
		mainWG.Add(1)
		go func() {
			defer mainWG.Done()

			logging.Info("starting http server", slog.Int("server number", i+1), slog.String("server url", cfg.Servers[i].URL))

			newSrv := httpserver.NewHTTPServer(cfg.Servers[i].URL, routes.DefaultRoutes())

			serversList = append(serversList, newSrv)

			startupMU.Lock()
			go func() {
				if cfg.ServersOutage.After != -1 {
					outAfter := outageAfter.Load().(float64)
					outageAfter.Store(float64(outAfter) * cfg.ServersOutage.Multiplier)
					fmt.Println("server", cfg.Servers[i].URL, "will stop after", time.Duration(outAfter)*time.Second)
					time.Sleep(time.Duration(outAfter) * time.Second)
					newSrv.Shutdown(context.Background())
				}
			}()
			startupMU.Unlock()

			if err := newSrv.Run(); err != nil {
				logging.Error("error runnning http server",
					slog.String("server url", cfg.Servers[i].URL),
					slog.String("error", err.Error()))
			}
		}()
	}

	// Заполнение массива доступных серверов для балансировщика.
	servers := make([]server.Server, len(cfg.Servers))
	for i := range cfg.Servers {
		servers[i] = *server.NewServer(
			cfg.Servers[i].URL,
			cfg.Servers[i].Weight,
		)
	}

	// Инициализация обработчика балансировщика.
	balancerHandler := balancer.NewBalancerHandler(logging, servers, cfg.BalancingAlg)

	// Запуск проверки статуса серверов.
	go func() {
		fmt.Println("starting health check")
		healthcheck.NewHealthChecker(logging, cfg.HealthCheck.Interval, cfg.HealthCheck.Timeout, balancerHandler.Balancer()).HealthCheck()
	}()

	// Инициализация балансировщика.
	srv := httpserver.NewHTTPServer(
		net.JoinHostPort(cfg.HTTPServer.Host, cfg.HTTPServer.Port),
		balancerHandler.Routes(),
	)

	// Запуск балансировщика отдельной горутиной.
	mainWG.Add(1)
	go func() {
		defer mainWG.Done()
		logging.Info("starting balancer http server", slog.String("server url", net.JoinHostPort(cfg.HTTPServer.Host, cfg.HTTPServer.Port)))

		if err := srv.Run(); err != nil {
			logging.Error("error runnning balancer http server",
				slog.String("server url", net.JoinHostPort(cfg.HTTPServer.Host, cfg.HTTPServer.Port)),
				slog.String("error", err.Error()))
		}
	}()

	newTlsSrv := httpserver.NewHTTPServerWithTLS(
		net.JoinHostPort(cfg.HTTPSServer.Host, cfg.HTTPSServer.Port),
		cfg.HTTPSServer.CertFile,
		cfg.HTTPSServer.KeyFile,
		balancerHandler.Routes(),
	)

	mainWG.Add(1)
	go func() {
		defer mainWG.Done()

		logging.Info("starting https server", slog.String("server url", net.JoinHostPort(cfg.HTTPSServer.Host, cfg.HTTPSServer.Port)))

		if err := newTlsSrv.Run(); err != nil {
			logging.Error("error runnning https server",
				slog.String("server url", net.JoinHostPort(cfg.HTTPSServer.Host, cfg.HTTPSServer.Port)),
				slog.String("error", err.Error()))
		}
	}()

	go func() {
		http.Handle("/metrics", promhttp.Handler())
		logging.Info("starting prometheus server", slog.String("server url", ":9091"))
		http.ListenAndServe(":9091", nil)
	}()

	// Ожидание завершения всех горутин.
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	logging.Info("shutting down...")

	for _, srv := range serversList {
		srv.Shutdown(context.Background())
	}

	newTlsSrv.Shutdown(context.Background())
	srv.Shutdown(context.Background())

	mainWG.Wait()

	logging.Info("shutdown complete")
}

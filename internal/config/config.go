package config

import (
	"log"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

const (
	defaultConfigsPath = "configs/config.yaml"
)

type Config struct {
	HTTPServer    HTTP         `yaml:"http_server"`
	HTTPSServer   HTTPS        `yaml:"https_server"`
	Servers       []Server     `yaml:"servers"`       // list of servers to connect to
	BalancingAlg  string       `yaml:"balancing_alg"` // balancing algorithm to use
	HealthCheck   Health       `yaml:"health_check"`
	Logging       Logging      `yaml:"logging"`
	ServersOutage ServerOutage `yaml:"servers_outage"`
}

type HTTP struct {
	Host string `yaml:"host"` // host to listen on
	Port string `yaml:"port"` // port to listen on
}

type HTTPS struct {
	Host     string `yaml:"host"` // host to listen on
	Port     string `yaml:"port"` // port to listen on
	CertFile string `yaml:"cert_file"`
	KeyFile  string `yaml:"key_file"`
}

type Server struct {
	URL    string `yaml:"url"`                    // url of the server
	Weight int    `yaml:"weight" env-default:"1"` // weight of the server
}

type Health struct {
	Interval time.Duration `yaml:"interval"` // interval between health checks
	Timeout  time.Duration `yaml:"timeout"`  // timeout for health checks (optional. default: 2s)        `yaml:"timeout"`  // timeout for health checks
}

type Logging struct {
	Rewrite bool   `yaml:"rewrite"`  // rewrite log file after startup or not
	Level   string `yaml:"level"`    // logging level
	Path    string `yaml:"path"`     // path to log file
	File    string `yaml:"file"`     // file to log to
	FileTLS string `yaml:"file_tls"` // file to log tls to
}

type ServerOutage struct {
	After      float64 `yaml:"after"`      // 'how many' seconds to wait till server outage
	Multiplier float64 `yaml:"multiplier"` // 'how much times' to mutiply time after last outage
}

func NewConfig(configPath string) *Config {
	var cfg Config

	if configPath == "" {
		log.Printf("config path is empty, using default path: %s", defaultConfigsPath)
		configPath = defaultConfigsPath
	}

	if err := cleanenv.ReadConfig(configPath, &cfg); err != nil {
		log.Fatalf("error reading config file: %s", err)
	}

	return &cfg
}

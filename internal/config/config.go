package config

import (
	"log"

	"github.com/ilyakaznacheev/cleanenv"
)

const (
	configPath = "configs/config.yaml"
)

type Config struct {
	HTTPServer   HTTP     `yaml:"http_server"`   // http server config
	Servers      []Server `yaml:"servers"`       // list of servers to connect to
	BalancingAlg string   `yaml:"balancing_alg"` // balancing algorithm to use
	HealthCheck  Health   `yaml:"health_check"`  // health check config
	Logging      Logging  `yaml:"logging"`       // logging config
}

type HTTP struct {
	Host string `yaml:"host"` // host to listen on
	Port string `yaml:"port"` // port to listen on
}

type Server struct {
	URL    string `yaml:"url"`                    // url of the server
	Weight int    `yaml:"weight" env-default:"1"` // weight of the server
}

type Health struct {
	Interval string `yaml:"interval"` // interval between health checks
	Timeout  string `yaml:"timeout"`  // timeout for health checks
}

type Logging struct {
	Level string `yaml:"level"` // logging level
	Path  string `yaml:"path"`  // path to log file
}

func NewConfig() *Config {
	var cfg Config

	if err := cleanenv.ReadConfig(configPath, &cfg); err != nil {
		log.Fatalf("error reading config file: %s", err)
	}

	return &cfg
}

package main

import (
	"fmt"

	"github.com/dzhordano/balancer-go/internal/config"
)

func main() {
	cfg := config.NewConfig()

	fmt.Println(cfg)

}

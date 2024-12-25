package main

import (
	"flag"
	"fmt"

	"github.com/dzhordano/balancer-go/internal/httpserver"
	"github.com/dzhordano/balancer-go/internal/routes"
)

func main() {
	var url string

	flag.StringVar(&url, "url", "", "url of server to start")
	flag.Parse()

	if url == "" {
		panic("url is empty. use flag -url")
	}

	srv := httpserver.NewHTTPServer(url, routes.DefaultRoutes())

	fmt.Println("starting server on", url)

	if err := srv.Run(); err != nil {
		panic(err)
	}

}

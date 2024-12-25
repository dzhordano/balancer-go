package main

import (
	"flag"
	"fmt"

	"github.com/dzhordano/balancer-go/internal/handler"
	"github.com/dzhordano/balancer-go/internal/server"
)

func main() {
	var url string

	flag.StringVar(&url, "url", "", "url of server to start")
	flag.Parse()

	if url == "" {
		panic("url is empty. use flag -url")
	}

	srv := server.NewHTTPServer(url, handler.DefaultRoutes())

	fmt.Println("starting server on", url)

	if err := srv.Run(); err != nil {
		panic(err)
	}

}

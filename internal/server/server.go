package server

import (
	"context"
	"net/http"
)

type httpServer struct {
	httpServer *http.Server
}

func NewHTTPServer(addr string, handler http.Handler) *httpServer {
	return &httpServer{
		httpServer: &http.Server{
			Addr:    addr,
			Handler: handler,
		},
	}
}

func (s *httpServer) Run() error {
	return s.httpServer.ListenAndServe()
}

func (s *httpServer) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}

package server

import (
	"context"
	"crypto/tls"
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

func NewHTTPServerWithTLS(addr, certFile, keyFile string, handler http.Handler) *httpServer {
	cfg := &tls.Config{
		MinVersion:               tls.VersionTLS12,
		CurvePreferences:         []tls.CurveID{tls.CurveP521, tls.CurveP384, tls.CurveP256},
		PreferServerCipherSuites: true,
		CipherSuites: []uint16{
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
			tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_RSA_WITH_AES_256_CBC_SHA,
		},
	}

	return &httpServer{
		httpServer: &http.Server{
			Addr:      addr,
			Handler:   handler,
			TLSConfig: cfg,
		},
	}
}

func (s *httpServer) Run() error {
	return s.httpServer.ListenAndServe()
}

func (s *httpServer) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}

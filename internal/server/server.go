package server

import "sync/atomic"

type Server struct {
	URL               string
	ActiveConnections int64
	Weight            int
}

func (s *Server) IncrementConnections() {
	atomic.AddInt64(&s.ActiveConnections, 1)
}

func (s *Server) DecrementConnections() {
	atomic.AddInt64(&s.ActiveConnections, -1)
}

func (s *Server) CurrentConnections() int64 {
	return atomic.LoadInt64(&s.ActiveConnections)
}

func NewServer(url string, weight int) *Server {
	return &Server{
		URL:    url,
		Weight: weight,
	}
}

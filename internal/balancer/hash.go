package balancer

import (
	"hash/crc32"
	"sync"

	"github.com/dzhordano/balancer-go/internal/server"
)

type HashBalancer struct {
	downServers  []server.Server
	aliveServers []server.Server
	mu           sync.Mutex
}

func (hb *HashBalancer) SetServers(servers []server.Server) {
	hb.mu.Lock()
	defer hb.mu.Unlock()
	hb.aliveServers = servers
}

func (hb *HashBalancer) SelectServer(args ...interface{}) *server.Server {
	hb.mu.Lock()
	defer hb.mu.Unlock()

	if len(hb.aliveServers) == 0 {
		return nil
	}

	if len(args) < 1 {
		return nil // FIXME Требуется ключ
	}

	key, ok := args[0].(string)
	if !ok {
		return nil
	}

	hash := crc32.ChecksumIEEE([]byte(key))
	index := int(hash) % len(hb.aliveServers)

	return &hb.aliveServers[index]
}

func (hb *HashBalancer) DownServers() []server.Server {
	hb.mu.Lock()
	defer hb.mu.Unlock()
	return hb.downServers
}

func (hb *HashBalancer) RemoveAliveServer(index int) {
	hb.mu.Lock()
	defer hb.mu.Unlock()
	hb.aliveServers = append(hb.aliveServers[:index], hb.aliveServers[index+1:]...)
}

func (hb *HashBalancer) RemoveDownServer(index int) {
	hb.mu.Lock()
	defer hb.mu.Unlock()
	hb.downServers = append(hb.downServers[:index], hb.downServers[index+1:]...)
}

func (hb *HashBalancer) AddAliveServer(server server.Server) {
	hb.mu.Lock()
	defer hb.mu.Unlock()
	hb.aliveServers = append(hb.aliveServers, server)
}

func (hb *HashBalancer) AddDownServer(server server.Server) {
	hb.mu.Lock()
	defer hb.mu.Unlock()
	hb.downServers = append(hb.downServers, server)
}

func (hb *HashBalancer) AliveServers() []server.Server {
	hb.mu.Lock()
	defer hb.mu.Unlock()
	return hb.aliveServers
}

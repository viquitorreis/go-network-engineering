package main

import (
	"net"
	"sync"
	"time"
)

type ServerConfig struct {
	Addr        string        // endereço para bind, ex: ":8080"
	Workers     int           // número de workers no pool
	QueueSize   int           // capacidade da fila de conexões
	IdleTimeout time.Duration // timeout para conexões idle
}

// Connection wraps net.Conn com métodos auxiliares
type Connection struct {
	net.Conn
	// TODO: adicionar campos úteis como ID, timestamp, etc
}

// Handler processa uma conexão. Retorna erro se algo der errado.
type Handler func(*Connection) error

// TCPServer gerencia o worker pool e a fila de conexões
type TCPServer struct {
	config   ServerConfig
	handler  Handler
	listener net.Listener
	connCh   chan *Connection // fila de conexões pendentes
	wg       sync.WaitGroup
	// TODO: campos para shutdown graceful
}

func NewTCPServer(config ServerConfig, handler Handler) *TCPServer {
	// TODO: inicializar server
	panic("not implemented")
}

// Start inicia o servidor — accept loop + worker pool
func (s *TCPServer) Start() error {
	// TODO:
	// 1. net.Listen no config.Addr
	// 2. Spawnar workers (s.config.Workers goroutines)
	// 3. Accept loop: aceita conexões e tenta enfileirar em s.connCh
	//    - Se connCh estiver cheio, rejeita a conexão (close imediato)
	// 4. Cada worker pega conexões de s.connCh e chama s.handler
	panic("not implemented")
}

// Shutdown para o servidor gracefully
func (s *TCPServer) Shutdown() error {
	// TODO:
	// 1. Fecha s.listener (para de aceitar novas conexões)
	// 2. Fecha s.connCh (workers terminam quando drenam a fila)
	// 3. s.wg.Wait() até todos os workers terminarem
	panic("not implemented")
}

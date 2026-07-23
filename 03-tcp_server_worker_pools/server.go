package main

import (
	"errors"
	"log/slog"
	"math/rand"
	"net"
	"sync"
	"time"
)

type ServerConfig struct {
	Addr        string
	Workers     int
	QueueSize   int
	IdleTimeout time.Duration
}

type Connection struct {
	net.Conn
	ID        int
	startedAt time.Time
}

type Handler func(*Connection) error

type TCPServer struct {
	config   ServerConfig
	handler  Handler
	listener net.Listener
	connCh   chan *Connection

	ready chan struct{}
	wg    sync.WaitGroup
	mu    sync.RWMutex
}

func NewTCPServer(config ServerConfig, handler Handler) *TCPServer {
	return &TCPServer{
		config:  config,
		handler: handler,
		connCh:  make(chan *Connection, config.QueueSize),
		ready:   make(chan struct{}),
	}
}

func (s *TCPServer) Start() error {
	listener, err := net.Listen("tcp", s.config.Addr)
	if err != nil {
		slog.Error("error creating new TCP server", "error", err)
		return err
	}

	s.mu.Lock()
	s.listener = listener
	s.mu.Unlock()

	for range s.config.Workers {
		s.wg.Go(func() {
			// cada worker vai ficar bloqueado aqui esperando novas conexões
			for conn := range s.connCh {
				s.handler(conn)
			}
		})
	}

	close(s.ready)

	for {
		conn, err := listener.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return nil
			}

			slog.Error("error accepting new conn", "error", err)
			continue
		}

		newConn := &Connection{
			Conn:      conn,
			ID:        rand.Int(),
			startedAt: time.Now(),
		}

		select {
		case s.connCh <- newConn:
		default:
			slog.Info("server is out of capacity")
			conn.Close()
		}
	}
}

func (s *TCPServer) Addr() string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.listener == nil {
		return ""
	}
	return s.listener.Addr().String()
}

func (s *TCPServer) Shutdown() error {
	var err error

	// tentamos fechar o listener, caso falhar por ter sido fechado antes externamente ou erro de I/O
	// continuamos fechando (best effort) o resto dos recursos para não deixar o servidor inconsistente
	s.mu.RLock()
	if closeErr := s.listener.Close(); closeErr != nil {
		err = closeErr
	}
	s.mu.RUnlock()

	close(s.connCh)
	s.wg.Wait()

	return err
}

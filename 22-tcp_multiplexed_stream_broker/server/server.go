package main

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net"
	"strconv"
	"sync"
)

type Server struct {
	listener *net.Listener
	broker   *Broker
	conns    map[net.Conn]*Client
	port     uint16

	ctx context.Context
	mu  sync.Mutex
	wg  sync.WaitGroup
}

type Commands [3]byte

var (
	SUBCmd Commands = [3]byte{'S', 'U', 'B'}
	PUBCmd Commands = [3]byte{'P', 'U', 'B'}
)

func NewServer(ctx context.Context, port uint16) *Server {
	return &Server{
		listener: nil,
		broker:   NewBroker(ctx),
		conns:    make(map[net.Conn]*Client),
		port:     port,
		ctx:      ctx,
	}
}

func (s *Server) Boostrap(ctx context.Context) {
	listener, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", s.port))
	if err != nil {
		log.Fatalf("err starting server: %v", err)
	}
	s.listener = &listener

	go func() {
		<-ctx.Done()
		slog.Info("shutting down listener")
		listener.Close()
	}()

	for {
		conn, err := listener.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				slog.Info("server context closed", "msg", "bye")
				return
			default:
				slog.Error("error accepting connection", "error", err.Error())
				continue
			}
		}

		// Set TCP_NODELAY to true to disable Nagle's algorithm for low-latency communication
		if tc, ok := conn.(*net.TCPConn); ok {
			tc.SetNoDelay(true)
		}

		go s.handleConn(conn)
	}
}

func (s *Server) AddToBroker(c *Client, t Topic) {
	s.broker.register <- Register{c, t}
}

func (s *Server) handleConn(conn net.Conn) {
	s.mu.Lock()
	client := NewClient(&conn)
	s.conns[conn] = client
	s.mu.Unlock()

	go s.handleRead(client)
}

func (s *Server) handleRead(c *Client) {
	errCh := make(chan error, 1)
	s.wg.Go(func() {
		for {
			select {
			case <-s.ctx.Done():
				return
			default:
				// buffer não pode ser fixo, tem que vir do length-prefix da própria msg
				buf := make([]byte, 4)
				_, err := io.ReadFull(*c.conn, buf)
				if err != nil && !errors.Is(err, io.EOF) {
					slog.Error(ErrReadingFromConn.Error(), "error", err.Error())
					errCh <- ErrReadingFromConn
					return
				}

				fmt.Println("size will be: ", string(buf))

				size := binary.BigEndian.Uint32(buf)
				buf = make([]byte, size)
				_, err = io.ReadFull(*c.conn, buf)
				if err != nil && !errors.Is(err, io.EOF) {
					slog.Error(ErrReadingFromConn.Error(), "error", err.Error())
					errCh <- ErrReadingFromConn
					return
				}

				fmt.Println("msg read:", string(buf))

				s.routeCommand(buf, c)
			}
		}
	})

	s.wg.Go(func() {
		select {
		case <-errCh:
			c.WriteFrame([]byte(ErrReadingFromConn.Error()))
			s.CloseClient(c)
		case <-s.ctx.Done():
			return
		}
	})
}

func (s *Server) CloseClient(c *Client) {
	s.mu.Lock()
	delete(s.conns, *c.conn)
	s.mu.Unlock()

	(*c.conn).Close()
}

func (s *Server) routeCommand(msg []byte, c *Client) {
	parts := bytes.Split(msg, []byte(" "))
	if len(parts) == 0 {
		return
	}

	if len(parts) == 0 || len(parts[0]) != 3 {
		slog.Error(ErrInvalidPayload.Error(), "error", ErrInvalidPayload.Error())
		c.WriteFrame([]byte(ErrInvalidPayload.Error()))
		return
	}
	cmd := Commands(parts[0])

	if len(parts) < 2 {
		slog.Error(ErrInvalidPayload.Error(), "error", ErrInvalidPayload.Error())
		c.WriteFrame([]byte(ErrInvalidPayload.Error()))
		return
	}

	topic := Topic(parts[1])
	if !topic.IsValid() {
		slog.Error(ErrInvalidTopic.Error(), "error", string(topic))
		c.WriteFrame([]byte(ErrInvalidTopic.Error()))
		return
	}

	switch cmd {
	case SUBCmd:
		s.broker.register <- Register{
			client: c,
			topic:  topic,
		}

	case PUBCmd:
		if len(parts) <= 2 {
			slog.Error(ErrInvalidPayload.Error(), "error", ErrInvalidPayload.Error())
			c.WriteFrame([]byte(ErrInvalidPayload.Error()))
			return
		}

		_, err := strconv.ParseFloat(string(parts[2]), 64)
		if err != nil {
			slog.Error(ErrInvalidPayload.Error(), "error", ErrInvalidPayload.Error())
			c.WriteFrame([]byte(ErrInvalidPayload.Error()))
			return
		}

		s.broker.broadcast <- &Message{
			topic: topic,
			data:  parts[2],
		}
	}
}

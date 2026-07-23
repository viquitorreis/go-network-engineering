package main

import (
	"context"
	"log"
	"log/slog"
	"math/rand"
	"net"
	"os"
	"os/signal"
	"syscall"
	"udpreliable/types"
)

func main() {
	addr, err := net.ResolveUDPAddr("udp4", ":8080")
	if err != nil {
		log.Fatalf("err resolving udp addr: %v", err)
	}

	ln, err := net.ListenUDP("udp4", addr)
	if err != nil {
		log.Fatalf("err resolving opening udp conn: %v", err)
	}
	defer ln.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	server := NewServer(ln, ctx, 0.1)

	log.Println("server is up and running")

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)

	go func() {
		<-sigCh
		cancel()
		server.Conn.Close()
	}()

	go server.Read()

	<-ctx.Done()
	log.Println("bye")
}

type Server struct {
	Conn     *net.UDPConn
	Msgs     map[string]map[uint64]types.Message
	Ctx      context.Context
	LossRate float64 // 0.0 a 1.0, to simulate, aplied on request and on ACK
}

func NewServer(conn *net.UDPConn, ctx context.Context, lossRate float64) *Server {
	return &Server{
		Conn:     conn,
		Msgs:     make(map[string]map[uint64]types.Message),
		Ctx:      ctx,
		LossRate: lossRate,
	}
}

func (s *Server) Read() {
	for {
		select {
		case <-s.Ctx.Done():
			return
		default:
		}

		buf := make([]byte, 512)
		n, addr, err := (*s.Conn).ReadFromUDP(buf)
		if err != nil {
			slog.Error("err reading from udp socket: %v", "error", err)
			break
		}

		// simulating request Loss
		if rand.Float64() < s.LossRate {
			continue
		}

		msg, err := types.ParseMsg(buf[:n])
		if err != nil {
			slog.Error("err parsing msg", "error", err)
			continue
		}

		if msg.Cmd != types.MSGCmd {
			continue
		}

		key := addr.String()
		if s.Msgs[key] == nil {
			s.Msgs[key] = make(map[uint64]types.Message)
		}

		if _, seen := s.Msgs[key][msg.ACK]; !seen {
			// first time seen it, process the message
			s.Msgs[key][msg.ACK] = *msg
			log.Printf("processando seq=%d de %s: %s", msg.ACK, key, msg.Content)
		}

		// if seen before, doesnt process the message, but still send the ACK

		s.Write(msg.ACK, addr)
	}
}

func (s *Server) Write(seq uint64, addr *net.UDPAddr) {
	// simulating ACK loss
	if rand.Float64() < s.LossRate {
		return
	}

	ack := types.Message{
		Cmd:     types.ACKCmd,
		Content: []byte("SEQ"),
		ACK:     seq,
	}
	datagram := ack.ToDatagram()

	if _, err := s.Conn.WriteToUDP(datagram, addr); err != nil {
		slog.Error("err writing ack", "error", err)
	}
}

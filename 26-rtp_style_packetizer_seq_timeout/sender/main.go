package main

import (
	"context"
	"encoding/binary"
	"log"
	"log/slog"
	"math/rand"
	"net"
	"os"
	"os/signal"
	"rtp_packetizer/protocol"
	"syscall"
	"time"
)

func main() {
	addr, err := net.ResolveUDPAddr("udp4", "localhost:8080")
	if err != nil {
		log.Fatalf("err resolving addr: %v", err)
	}

	udpConn, err := net.DialUDP("udp4", nil, addr)
	if err != nil {
		log.Fatalf("err opening udp conn: %v", err)
	}
	defer udpConn.Close()

	log.Println("sender up and running")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		log.Println("bye")
	}()

	// 20 ms interval sender
	sender := NewSender(ctx, udpConn, 20)
	go sender.Send()

	<-ctx.Done()
}

type Sender struct {
	ctx        context.Context
	Conn       net.Conn
	LastSeqNum uint32
	Interval   time.Duration
}

func NewSender(ctx context.Context, conn net.Conn, interval int) *Sender {
	return &Sender{
		ctx:        ctx,
		Conn:       conn,
		LastSeqNum: 0,
		Interval:   time.Millisecond * time.Duration(interval),
	}
}

func (s *Sender) Send() {
	ticker := time.NewTicker(s.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			msg := s.Generate()

			if rand.Float64() < 0.15 {
				continue
			}

			payload := msg.ToDatagram()

			buf := make([]byte, 4)
			binary.LittleEndian.PutUint32(buf, uint32(len(payload)))

			if _, err := s.Conn.Write(buf); err != nil {
				slog.Error("error writing buf size", "error", err)
				return
			}

			if _, err := s.Conn.Write(msg.ToDatagram()); err != nil {
				slog.Error("error writing", "error", err.Error())
				return
			}
		}
	}
}

func (s *Sender) Generate() protocol.RTPDatagram {
	msg := protocol.RTPDatagram{
		Payload:   uint16(rand.Uint32()),
		SeqNumber: s.LastSeqNum,
		Timestamp: uint64(time.Now().Unix()),
	}

	s.LastSeqNum++

	return msg
}

package main

import (
	"context"
	"encoding/binary"
	"errors"
	"io"
	"log"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"rtp_packetizer/protocol"
	"syscall"
)

func main() {
	udpAddr, err := net.ResolveUDPAddr("udp4", "localhost:8080")
	if err != nil {
		log.Fatalf("err resolving udp addr: %v", err)
	}

	udpConn, err := net.ListenUDP("udp4", udpAddr)
	if err != nil {
		log.Fatalf("err opening udp conn: %v", err)
	}
	defer udpConn.Close()

	log.Println("receptor up and running")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		cancel()
		log.Println("bye")
	}()

	receptor := NewReceptor(ctx, udpConn)

	go receptor.handleRead()

	<-ctx.Done()
}

type Receptor struct {
	ctx         context.Context
	Conn        net.Conn
	LastSeqNum  uint32
	initialized bool
	Delayed     []*protocol.RTPDatagram
	SeqGaps     []*protocol.RTPDatagram // sequence gaps
}

func NewReceptor(ctx context.Context, conn net.Conn) *Receptor {
	return &Receptor{
		ctx:        ctx,
		Conn:       conn,
		LastSeqNum: 0,
		Delayed:    make([]*protocol.RTPDatagram, 0),
		SeqGaps:    make([]*protocol.RTPDatagram, 0),
	}
}

func (r *Receptor) handleRead() {
	for {
		select {
		case <-r.ctx.Done():
			return
		default:
		}

		buf := make([]byte, 4)
		_, err := io.ReadFull(r.Conn, buf)
		if err != nil {
			slog.Error("error reading buf size", "error", err)
			return
		}

		size := binary.LittleEndian.Uint32(buf)

		buf = make([]byte, size)
		_, err = io.ReadFull(r.Conn, buf)
		if err != nil && !errors.Is(err, io.EOF) {
			slog.Error("err reading from conn", "error", err.Error())
			return
		}

		msg := protocol.Parse(buf)
		log.Println("received: ", msg)

		if !r.initialized {
			r.LastSeqNum = msg.SeqNumber
			r.initialized = true
		} else if msg.SeqNumber > r.LastSeqNum+1 {
			r.SeqGaps = append(r.SeqGaps, msg)
			log.Printf("message gap: %d packets lost before seq %d", len(r.SeqGaps), msg.SeqNumber)
			r.LastSeqNum = msg.SeqNumber
		} else if msg.SeqNumber <= r.LastSeqNum {
			log.Printf("packet out of order or late: seq %d (seen until: %d)", msg.SeqNumber, r.LastSeqNum)
			r.Delayed = append(r.Delayed, msg)
		} else {
			r.LastSeqNum = msg.SeqNumber
		}
	}
}

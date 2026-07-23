package main

import (
	"context"
	"encoding/binary"
	"errors"
	"io"
	"jitter_buffer/protocol"
	"log"
	"log/slog"
	"net"
	"sync"
	"time"
)

type Receptor struct {
	ctx         context.Context
	Conn        net.Conn
	Jb          *JitterBuffer
	LastSeqNum  uint64
	initialized bool
	Delayed     []*protocol.RTPDatagram
	SeqGaps     []*protocol.RTPDatagram // sequence gaps

	mu sync.Mutex
}

func NewReceptor(ctx context.Context, conn net.Conn) *Receptor {
	return &Receptor{
		ctx:        ctx,
		Conn:       conn,
		Jb:         NewJitterBuffer(ctx, 20*time.Millisecond),
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

		// every packet that arises needs to pass throughout the JitterBuffer
		// its the buffer who decides the liberation ordem of packets based on timestamp, not the receptor itself
		packet := &Packet{
			Data:      uint64(msg.Payload),
			SeqNumber: msg.SeqNumber,
			Timestamp: msg.Timestamp,
		}
		r.Jb.Insert(msg.SeqNumber, packet)

		r.mu.Lock()
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
		r.mu.Unlock()
	}
}

package main

import (
	"context"
	"encoding/binary"
	"errors"
	"feq_parity_packets/protocol"
	"io"
	"log"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
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
	go receptor.monitor()

	<-ctx.Done()
}

type Receptor struct {
	ctx  context.Context
	Conn net.Conn
	// groupID -> group state
	groupStates map[uint64]groupState
	maxWait     time.Duration
	stats       Stats

	mu sync.Mutex
}

type groupState struct {
	// we need the real datagram in order to
	// calculate their XOR, and to see if its possible
	groupSeqs  []protocol.Datagram
	arrivedAt  time.Time
	haveParity bool
}

type Stats struct {
	Completed uint64
	Recovered uint64
	Lost      uint64
}

func NewReceptor(ctx context.Context, conn net.Conn) *Receptor {
	return &Receptor{
		ctx:         ctx,
		Conn:        conn,
		groupStates: make(map[uint64]groupState),
		maxWait:     time.Millisecond * 200,
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
		if msg == nil {
			slog.Error("failed to parse datagram, skipping")
			continue
		}

		log.Println("received: ", msg)

		r.mu.Lock()
		state := r.groupStates[msg.GroupID]
		state.groupSeqs = append(state.groupSeqs, *msg)

		if msg.PacketType == protocol.ParityPacket {
			state.haveParity = true
		}

		if state.arrivedAt.IsZero() {
			state.arrivedAt = time.Now()
		}

		r.groupStates[msg.GroupID] = state
		r.mu.Unlock()
	}
}

func (r *Receptor) monitor() {
	ticker := time.NewTicker(time.Millisecond * 20)
	defer ticker.Stop()

	for {
		select {
		case <-r.ctx.Done():
			return
		case <-ticker.C:
			r.mu.Lock()

			for groupID, state := range r.groupStates {
				if state.arrivedAt.IsZero() || time.Since(state.arrivedAt) < r.maxWait {
					continue // group is still on waiting time window
				}

				dataReceived := 0
				var (
					xorAcc uint64
					parity uint64
				)

				for _, d := range state.groupSeqs {
					if d.PacketType == protocol.DataPacket {
						dataReceived++
						xorAcc ^= d.Payload
					} else {
						parity = d.Payload
					}
				}

				const groupSize = 4
				missing := groupSize - dataReceived

				switch {
				case missing == 0:
					log.Printf("group: %d is complete, nothing to do", groupID)
					r.stats.Completed++
				case missing == 1 && state.haveParity:
					recovered := xorAcc ^ parity
					r.stats.Recovered++

					log.Printf("group: %d recovered using FEC, payload reconstructed: %d", groupID, recovered)

				default:
					r.stats.Lost++

					log.Printf("group %d: definite loss (%d of %d packages missing)", groupID, missing, groupSize)
				}

				delete(r.groupStates, groupID)
			}

			r.mu.Unlock()
		}
	}

}

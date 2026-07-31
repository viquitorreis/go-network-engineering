package main

import (
	"context"
	"feq_parity_packets/protocol"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net"
	"os"
	"testing"
	"time"
)

// simulateGroup sends one FEC group (groupSize data packets + 1 parity),
// each packet independently dropped with probability lossRate -- exactly
// mirroring how the real sender's loss simulation works.
func simulateGroup(conn net.Conn, groupID uint64, groupSize uint8, lossRate float64, r *rand.Rand) {
	payloads := make([]uint64, groupSize)
	var parity uint64
	for i := range payloads {
		payloads[i] = r.Uint64()
		parity ^= payloads[i]
	}

	for i, p := range payloads {
		if r.Float64() < lossRate {
			continue
		}
		conn.Write(buildFrame(groupID, uint64(i), p, 0, protocol.DataPacket))
	}
	if r.Float64() < lossRate {
		return
	}
	conn.Write(buildFrame(groupID, 0, parity, 0, protocol.ParityPacket))
}

// BenchmarkFEC_RecoveryVsGroupSize measures, across group sizes and loss
// rates, what fraction of groups end up recovered vs. definitively lost --
// the empirical version of the "bigger group = more double-loss exposure"
// trade-off.
func BenchmarkFEC_RecoveryVsGroupSize(b *testing.B) {
	log.SetOutput(io.Discard)
	defer log.SetOutput(os.Stderr)

	groupSizes := []uint8{4, 8, 16}
	lossRates := []float64{0.05, 0.10, 0.20}

	for _, gs := range groupSizes {
		for _, lr := range lossRates {
			name := fmt.Sprintf("group_%d_loss_%d%%", gs, int(lr*100))
			b.Run(name, func(b *testing.B) {
				serverSide, clientSide := net.Pipe()
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()

				r := NewReceptor(ctx, gs, serverSide)
				r.maxWait = 30 * time.Millisecond
				go r.handleRead()
				go r.monitor()

				rng := rand.New(rand.NewSource(42))

				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					simulateGroup(clientSide, uint64(i), gs, lr, rng)
				}
				b.StopTimer()

				time.Sleep(100 * time.Millisecond) // let monitor finish resolving the last groups

				r.mu.Lock()
				completed := r.stats.Completed
				recovered := r.stats.Recovered
				lost := r.stats.Lost
				r.mu.Unlock()

				total := completed + recovered + lost
				var recoveryRate, lossRate float64
				if total > 0 {
					recoveryRate = float64(recovered) / float64(total) * 100
					lossRate = float64(lost) / float64(total) * 100
				}
				bandwidthOverhead := 100.0 / float64(gs) // 1 parity packet per gs data packets

				b.ReportMetric(recoveryRate, "%recovered")
				b.ReportMetric(lossRate, "%definitive_loss")
				b.ReportMetric(bandwidthOverhead, "%bandwidth_overhead")
			})
		}
	}
}

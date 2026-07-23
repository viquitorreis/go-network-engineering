package main

import (
	"context"
	"fmt"
	"net"
	"sort"
	"testing"
	"time"
	"udpreliable/types"
)

// startBenchServer starts a real Server with a configurable loss rate,
// similar to startTestServer used in functional tests, but accepting *testing.B instead of *testing.T.
func startBenchServer(b *testing.B, lossRate float64) (addr *net.UDPAddr, cleanup func()) {
	b.Helper()

	laddr, err := net.ResolveUDPAddr("udp4", "127.0.0.1:0")
	if err != nil {
		b.Fatalf("err resolvendo addr de benchmark: %v", err)
	}
	conn, err := net.ListenUDP("udp4", laddr)
	if err != nil {
		b.Fatalf("err abrindo socket de benchmark: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	server := NewServer(conn, ctx, lossRate)
	go server.Read()

	cleanup = func() {
		cancel()
		conn.Close()
	}
	return conn.LocalAddr().(*net.UDPAddr), cleanup
}

// benchConfig replicates the same parameters of the real client (challenge 24):
// stop-and-wait, short timeout, maximum number of attempts.
const (
	benchMaxRetries  = 5
	benchReadTimeout = 100 * time.Millisecond // shorter than the real client (500ms) just to keep the benchmark fast
)

// reliableSend sends a message with the same protocol as the client (seq + wait for ACK + retransmit on timeout),
// and returns how many attempts were needed and whether it finished confirmed or not.
// Reimplemented here (instead of importing the Client) because client
// and server are separate "main" packages and are not importable between each other.
func reliableSend(conn *net.UDPConn, seq uint64, text string) (attempts int, ok bool) {
	msg := types.Message{Cmd: types.MSGCmd, Content: []byte(text), ACK: seq}
	datagram := msg.ToDatagram()

	for attempts = 1; attempts <= benchMaxRetries; attempts++ {
		if _, err := conn.Write(datagram); err != nil {
			return attempts, false
		}

		conn.SetReadDeadline(time.Now().Add(benchReadTimeout))
		buf := make([]byte, 512)
		n, err := conn.Read(buf)
		if err != nil {
			continue // timeout -- tenta de novo
		}

		reply, err := types.ParseMsg(buf[:n])
		if err != nil || reply.Cmd != types.ACKCmd || reply.ACK != seq {
			continue
		}
		return attempts, true
	}
	return attempts - 1, false
}

// percentile assumes latencies is already sorted.
func percentile(sorted []time.Duration, p float64) time.Duration {
	if len(sorted) == 0 {
		return 0
	}
	idx := int(float64(len(sorted)-1) * p)
	return sorted[idx]
}

// BenchmarkReliability runs a benchmark to measure the reliability of the UDP server under
// different packet loss rates. It simulates sending messages with a stop-and-wait protocol,
// measuring retry rates, failure rates, and latency percentiles (p50 and p99) for each loss rate.
//
// Run with `go test -bench=BenchmarkReliability -benchtime=200x ./...`
// (-benchtime=Nx runs exactly N messages per loss rate, instead of letting Go calibrate automatically
// which is more predictable for comparing different rates with the same sample size)
func BenchmarkReliability(b *testing.B) {
	lossRates := []float64{0.0, 0.05, 0.10, 0.20, 0.40}

	for _, lossRate := range lossRates {
		lossRate := lossRate

		b.Run(fmt.Sprintf("loss_%d%%", int(lossRate*100)), func(b *testing.B) {
			addr, cleanup := startBenchServer(b, lossRate)
			defer cleanup()

			conn, err := net.DialUDP("udp4", nil, addr)
			if err != nil {
				b.Fatalf("err discando pro server de benchmark: %v", err)
			}
			defer conn.Close()

			latencies := make([]time.Duration, 0, b.N)
			retried := 0
			failed := 0

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				seq := uint64(i + 1)

				start := time.Now()
				attempts, ok := reliableSend(conn, seq, "benchmark payload")
				elapsed := time.Since(start)

				latencies = append(latencies, elapsed)
				if attempts > 1 {
					retried++
				}

				if !ok {
					failed++
				}
			}
			b.StopTimer()

			sort.Slice(latencies, func(i, j int) bool { return latencies[i] < latencies[j] })

			retryRate := float64(retried) / float64(b.N) * 100
			failureRate := float64(failed) / float64(b.N) * 100
			p50 := percentile(latencies, 0.50)
			p99 := percentile(latencies, 0.99)

			b.ReportMetric(retryRate, "%retry")
			b.ReportMetric(failureRate, "%fail")
			b.ReportMetric(float64(p50.Milliseconds()), "p50_ms")
			b.ReportMetric(float64(p99.Milliseconds()), "p99_ms")
		})
	}
}
